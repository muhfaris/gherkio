package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/muhfaris/gherkio/internal/loader"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import curl commands or OpenAPI specs into catalogs/fixtures",
	Long:  `Imports API definitions from curl commands or OpenAPI specifications, generating catalogs and fixtures.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImport(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().String("api", "", `API key to generate (e.g. "users.create")`)
	importCmd.Flags().String("curl", "", `Raw curl command to import (e.g. 'curl -X POST ...')`)
	importCmd.Flags().String("catalog", "", "Catalog file to update/create (defaults to gherkio/apis/imported.yaml)")
	importCmd.Flags().String("fixture", "", "Fixture file to write (optional)")
	importCmd.Flags().String("feature", "", "Feature file to write (optional)")
	importCmd.Flags().String("title", "", "Scenario title (optional)")
	importCmd.Flags().String("name", "", "Feature name (optional)")
	importCmd.Flags().String("openapi", "", "Path to OpenAPI file (YAML/JSON) to import")
	importCmd.Flags().String("fixtures", "", "Directory for generated fixtures in OpenAPI mode (defaults to gherkio/fixtures/openapi)")
	importCmd.Flags().String("prefix", "", "Optional prefix for generated API keys in OpenAPI mode")
}

type curlImportOptions struct {
	APIKey     string
	CurlCmd    string
	Catalog    string
	Fixture    string
	Feature    string
	Title      string
	ReportName string
}

type openapiImportOptions struct {
	SpecPath    string
	CatalogPath string
	FixturesDir string
	KeyPrefix   string
}

func runImport(cmd *cobra.Command, args []string) error {
	apiKey, _ := cmd.Flags().GetString("api")
	curlCmd, _ := cmd.Flags().GetString("curl")
	catalog, _ := cmd.Flags().GetString("catalog")
	fixture, _ := cmd.Flags().GetString("fixture")
	feature, _ := cmd.Flags().GetString("feature")
	title, _ := cmd.Flags().GetString("title")
	reportName, _ := cmd.Flags().GetString("name")
	openapiSpec, _ := cmd.Flags().GetString("openapi")
	openapiFixtures, _ := cmd.Flags().GetString("fixtures")
	openapiPrefix, _ := cmd.Flags().GetString("prefix")

	if openapiSpec != "" {
		if curlCmd != "" || apiKey != "" || fixture != "" || feature != "" || title != "" || reportName != "" {
			return errors.New("cannot combine --openapi with curl import flags like --curl or --api")
		}
		if catalog == "" {
			catalog = "gherkio/apis/openapi.yaml"
		}
		if openapiFixtures == "" {
			openapiFixtures = "gherkio/fixtures/openapi"
		}
		opts := openapiImportOptions{
			SpecPath:    openapiSpec,
			CatalogPath: catalog,
			FixturesDir: openapiFixtures,
			KeyPrefix:   openapiPrefix,
		}
		return runImportOpenAPIMode(opts)
	}

	if apiKey == "" {
		return errors.New("--api is required")
	}
	if curlCmd == "" {
		return errors.New("--curl is required")
	}
	if catalog == "" {
		catalog = "gherkio/apis/imported.yaml"
	}

	opts := curlImportOptions{
		APIKey:     apiKey,
		CurlCmd:    curlCmd,
		Catalog:    catalog,
		Fixture:    fixture,
		Feature:    feature,
		Title:      title,
		ReportName: reportName,
	}
	return runImportCurlMode(opts)
}

func runImportCurlMode(opts curlImportOptions) error {
	spec, err := parseCurlCommand(opts.CurlCmd)
	if err != nil {
		return fmt.Errorf("parse curl: %w", err)
	}

	req, err := buildReplayRequest(spec)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)

	fmt.Printf("HTTP %s %s -> %d\n", spec.Method, spec.URL, resp.StatusCode)
	preview := previewText(bodyBytes)
	if preview != "" {
		fmt.Printf("Response preview:%s\n", preview)
	}

	generateAssertions := false
	if json.Valid(bodyBytes) {
		generateAssertions = promptYesNo("Generate JSON assertions from response? (y/N): ")
	} else {
		fmt.Println("Response is not valid JSON; will only assert on status code.")
	}

	fixture := opts.Fixture
	if fixture == "" && spec.HasBody() {
		if spec.HasMultipart() {
			fixture = defaultMultipartFixturePath(opts.APIKey)
		} else {
			fixture = defaultFixturePath(opts.APIKey)
		}
	}
	feature := opts.Feature
	if feature == "" {
		feature = defaultFeaturePath(opts.APIKey)
	}

	title := opts.Title
	if title == "" {
		title = fmt.Sprintf("Call %s", opts.APIKey)
	}
	reportName := opts.ReportName
	if reportName == "" {
		reportName = humanizeAPIKey(opts.APIKey)
	}

	featureFixtureRef := trimFixtureReference(fixture)

	var (
		fixtureContent string
		fileCopies     []fileCopy
	)
	if spec.HasMultipart() {
		var err error
		fixtureContent, fileCopies, err = buildMultipartFixture(spec, fixture)
		if err != nil {
			return fmt.Errorf("build multipart fixture: %w", err)
		}
	} else {
		formattedBody := spec.Body
		if pretty, ok := prettyJSON(spec.Body); ok {
			formattedBody = pretty
		}
		fixtureContent = formattedBody
	}

	assertions := []string{fmt.Sprintf("Then response status should be %d", resp.StatusCode)}
	if generateAssertions {
		jsonAssertions := buildJSONAssertions(bodyBytes)
		if len(jsonAssertions) > 0 {
			assertions = append(assertions, jsonAssertions...)
		}
	}

	apiEntry := buildAPIEntry(opts.APIKey, spec)
	featureContent := buildFeatureContent(reportName, title, opts.APIKey, featureFixtureRef, spec.HasBody(), assertions)

	fmt.Println("\nPlanned outputs:")
	fmt.Printf("- Catalog entry: %s (key: %s)\n", opts.Catalog, opts.APIKey)
	if spec.HasBody() {
		fmt.Printf("- Fixture: %s\n", fixture)
	}
	fmt.Printf("- Feature: %s\n", feature)

	if !promptYesNo("Write files? (y/N): ") {
		fmt.Println("Aborted by user")
		return nil
	}

	if err := writeCatalogEntry(opts.Catalog, opts.APIKey, apiEntry); err != nil {
		return fmt.Errorf("write catalog: %w", err)
	}
	if spec.HasBody() {
		if err := writeFileEnsureDir(fixture, fixtureContent); err != nil {
			return fmt.Errorf("write fixture: %w", err)
		}
		for _, c := range fileCopies {
			if err := copyFileEnsureDir(c.Src, c.Dest); err != nil {
				return fmt.Errorf("copy %s -> %s: %w", c.Src, c.Dest, err)
			}
		}
	}
	if err := writeFileEnsureDir(feature, featureContent); err != nil {
		return fmt.Errorf("write feature: %w", err)
	}

	fmt.Println("Import completed.")
	return nil
}

func runImportOpenAPIMode(opts openapiImportOptions) error {
	if strings.TrimSpace(opts.SpecPath) == "" {
		return errors.New("--openapi is required")
	}

	data, err := os.ReadFile(opts.SpecPath)
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}

	doc, err := parseOpenAPIDoc(data)
	if err != nil {
		return fmt.Errorf("parse OpenAPI: %w", err)
	}

	items, err := buildOpenAPIImports(doc, opts.KeyPrefix, opts.FixturesDir)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Println("No operations found to import.")
		return nil
	}

	fmt.Printf("Found %d operations in %s\n", len(items), opts.SpecPath)
	fmt.Println("\nPlanned outputs:")
	for _, item := range items {
		if item.FixturePath != "" {
			fmt.Printf("- %s -> %s %s (fixture: %s)\n", item.Key, item.Method, item.Path, item.FixturePath)
		} else {
			fmt.Printf("- %s -> %s %s\n", item.Key, item.Method, item.Path)
		}
	}

	if !promptYesNo("Write files? (y/N): ") {
		fmt.Println("Aborted by user")
		return nil
	}

	catalogEntries := make(map[string]loader.Endpoint, len(items))
	for _, item := range items {
		catalogEntries[item.Key] = item.Endpoint
	}
	if err := writeCatalogEntries(opts.CatalogPath, catalogEntries); err != nil {
		return fmt.Errorf("write catalog: %w", err)
	}
	for _, item := range items {
		if item.FixturePath == "" || item.FixtureContent == "" {
			continue
		}
		if err := writeFileEnsureDir(item.FixturePath, item.FixtureContent); err != nil {
			return fmt.Errorf("write fixture %s: %w", item.FixturePath, err)
		}
	}

	fmt.Println("Import completed.")
	return nil
}

type curlSpec struct {
	Method    string
	URL       string
	Headers   map[string][]string
	Body      string
	FormParts []formPart
}

func (c curlSpec) HasBody() bool {
	return strings.TrimSpace(c.Body) != "" || len(c.FormParts) > 0
}

func (c curlSpec) HasMultipart() bool {
	return len(c.FormParts) > 0
}

type formPart struct {
	Name        string
	Value       string
	FilePath    string
	FileName    string
	ContentType string
	IsFile      bool
}

type openAPIDoc struct {
	Paths      map[string]openAPIPathItem `yaml:"paths"`
	Components openAPIComponents          `yaml:"components"`
}

type openAPIPathItem struct {
	Delete  *openAPIOperation `yaml:"delete"`
	Get     *openAPIOperation `yaml:"get"`
	Head    *openAPIOperation `yaml:"head"`
	Options *openAPIOperation `yaml:"options"`
	Patch   *openAPIOperation `yaml:"patch"`
	Post    *openAPIOperation `yaml:"post"`
	Put     *openAPIOperation `yaml:"put"`
	Trace   *openAPIOperation `yaml:"trace"`
}

type openAPIOperation struct {
	OperationID string              `yaml:"operationId"`
	Summary     string              `yaml:"summary"`
	Tags        []string            `yaml:"tags"`
	RequestBody *openAPIRequestBody `yaml:"requestBody"`
}

type openAPIRequestBody struct {
	Ref     string                      `yaml:"$ref"`
	Content map[string]openAPIMediaType `yaml:"content"`
	Desc    string                      `yaml:"description"`
	Req     bool                        `yaml:"required"`
}

type openAPIMediaType struct {
	Example  any                       `yaml:"example"`
	Examples map[string]openAPIExample `yaml:"examples"`
	Schema   *openAPISchema            `yaml:"schema"`
}

type openAPIExample struct {
	Value any `yaml:"value"`
}

type openAPISchema struct {
	Ref         string                    `yaml:"$ref"`
	Type        string                    `yaml:"type"`
	Format      string                    `yaml:"format"`
	Title       string                    `yaml:"title"`
	Description string                    `yaml:"description"`
	Properties  map[string]*openAPISchema `yaml:"properties"`
	Items       *openAPISchema            `yaml:"items"`
	Required    []string                  `yaml:"required"`
	Enum        []any                     `yaml:"enum"`
	AllOf       []*openAPISchema          `yaml:"allOf"`
	OneOf       []*openAPISchema          `yaml:"oneOf"`
	AnyOf       []*openAPISchema          `yaml:"anyOf"`
	Example     any                       `yaml:"example"`
	Default     any                       `yaml:"default"`
	Nullable    bool                      `yaml:"nullable"`
}

type openAPIComponents struct {
	Schemas       map[string]*openAPISchema      `yaml:"schemas"`
	RequestBodies map[string]*openAPIRequestBody `yaml:"requestBodies"`
}

type openAPIImportItem struct {
	Key            string
	Method         string
	Path           string
	Endpoint       loader.Endpoint
	FixturePath    string
	FixtureContent string
}

func parseOpenAPIDoc(data []byte) (*openAPIDoc, error) {
	var doc openAPIDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if len(doc.Paths) == 0 {
		return &doc, nil
	}
	return &doc, nil
}

func buildOpenAPIImports(doc *openAPIDoc, prefix, fixturesDir string) ([]openAPIImportItem, error) {
	if doc == nil || len(doc.Paths) == 0 {
		return nil, nil
	}
	prefix = strings.Trim(prefix, ".")
	if prefix != "" {
		prefix = sanitizeKey(prefix)
	}

	var items []openAPIImportItem
	seen := map[string]int{}

	pathKeys := make([]string, 0, len(doc.Paths))
	for p := range doc.Paths {
		pathKeys = append(pathKeys, p)
	}
	sort.Strings(pathKeys)

	for _, path := range pathKeys {
		item := doc.Paths[path]
		for method, op := range map[string]*openAPIOperation{
			"DELETE":  item.Delete,
			"GET":     item.Get,
			"HEAD":    item.Head,
			"OPTIONS": item.Options,
			"PATCH":   item.Patch,
			"POST":    item.Post,
			"PUT":     item.Put,
			"TRACE":   item.Trace,
		} {
			if op == nil {
				continue
			}
			keyBase := deriveOpenAPIKey(op, method, path)
			if keyBase == "" {
				keyBase = fmt.Sprintf("operation.%s.%s", strings.ToLower(method), sanitizeSegment(path))
			}
			if prefix != "" {
				keyBase = prefix + "." + keyBase
			}
			key := ensureUniqueKey(keyBase, seen)

			fixtureContent, fixturePath, err := generateOpenAPIFixture(doc, op, key, fixturesDir)
			if err != nil {
				return nil, fmt.Errorf("build fixture for %s: %w", key, err)
			}

			items = append(items, openAPIImportItem{
				Key:            key,
				Method:         method,
				Path:           path,
				Endpoint:       loader.Endpoint{Method: method, Path: path},
				FixturePath:    fixturePath,
				FixtureContent: fixtureContent,
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Key == items[j].Key {
			return items[i].Method < items[j].Method
		}
		return items[i].Key < items[j].Key
	})
	return items, nil
}

func deriveOpenAPIKey(op *openAPIOperation, method, rawPath string) string {
	if op != nil && strings.TrimSpace(op.OperationID) != "" {
		if key := sanitizeKey(op.OperationID); key != "" {
			return key
		}
	}

	var segments []string
	if op != nil {
		for _, tag := range op.Tags {
			if seg := sanitizeSegment(tag); seg != "" {
				segments = append(segments, seg)
				break
			}
		}
	}
	if len(segments) == 0 {
		for _, part := range strings.Split(strings.Trim(rawPath, "/"), "/") {
			if seg := sanitizeSegment(part); seg != "" {
				segments = append(segments, seg)
			}
		}
	}
	if len(segments) == 0 {
		segments = append(segments, "root")
	}
	segments = append(segments, strings.ToLower(method))
	return strings.Join(segments, ".")
}

func sanitizeKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", ".", "-", ".", "_", ".", "/", ".", ":", ".", "::", ".", "..", ".")
	raw = replacer.Replace(raw)
	parts := strings.Split(raw, ".")
	var cleaned []string
	for _, part := range parts {
		if seg := sanitizeSegment(part); seg != "" {
			cleaned = append(cleaned, seg)
		}
	}
	return strings.Join(cleaned, ".")
}

func sanitizeSegment(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "{}")
	if raw == "" {
		return ""
	}
	raw = strings.ReplaceAll(raw, "/", "_")
	raw = strings.ReplaceAll(raw, "-", "_")
	raw = strings.ReplaceAll(raw, ".", "_")
	raw = strings.ReplaceAll(raw, " ", "_")
	raw = strings.ToLower(raw)

	var b strings.Builder
	lastUnderscore := false
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := b.String()
	out = strings.Trim(out, "_")
	return out
}

func ensureUniqueKey(base string, counts map[string]int) string {
	if base == "" {
		base = "operation"
	}
	count := counts[base]
	if count == 0 {
		counts[base] = 1
		return base
	}
	for {
		count++
		candidate := fmt.Sprintf("%s_%d", base, count)
		if counts[candidate] == 0 {
			counts[base] = count
			counts[candidate] = 1
			return candidate
		}
	}
}

func generateOpenAPIFixture(doc *openAPIDoc, op *openAPIOperation, key, fixturesDir string) (string, string, error) {
	if doc == nil || op == nil || op.RequestBody == nil {
		return "", "", nil
	}
	resolvedBody, err := doc.resolveRequestBody(op.RequestBody, map[string]bool{}, 0)
	if err != nil {
		return "", "", err
	}
	if resolvedBody == nil || len(resolvedBody.Content) == 0 {
		return "", "", nil
	}
	mediaTypeName, media := resolvedBody.preferredMediaType()
	if media == nil {
		return "", "", nil
	}
	if !strings.HasPrefix(strings.ToLower(mediaTypeName), "application/json") {
		return "", "", nil
	}
	example, err := doc.exampleFromMediaType(media)
	if err != nil {
		return "", "", err
	}
	if example == nil {
		return "", "", nil
	}
	normalized := normalizeYAMLValue(example)
	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return "", "", err
	}
	if fixturesDir == "" {
		fixturesDir = "gherkio/fixtures/openapi"
	}
	path := fixturePathForKey(fixturesDir, key)
	return string(data), path, nil
}

func (rb *openAPIRequestBody) preferredMediaType() (string, *openAPIMediaType) {
	if rb == nil || len(rb.Content) == 0 {
		return "", nil
	}
	if mt, ok := rb.Content["application/json"]; ok {
		copy := mt
		return "application/json", &copy
	}
	for k, v := range rb.Content {
		copy := v
		return k, &copy
	}
	return "", nil
}

func (doc *openAPIDoc) resolveRequestBody(body *openAPIRequestBody, seen map[string]bool, depth int) (*openAPIRequestBody, error) {
	if body == nil {
		return nil, nil
	}
	if depth > 16 {
		return nil, errors.New("requestBody reference depth exceeded")
	}
	if body.Ref == "" {
		return body, nil
	}
	name, ok := strings.CutPrefix(body.Ref, "#/components/requestBodies/")
	if !ok {
		return nil, fmt.Errorf("unsupported requestBody ref: %s", body.Ref)
	}
	if seen[name] {
		return nil, nil
	}
	seen[name] = true
	if doc.Components.RequestBodies == nil {
		return nil, fmt.Errorf("requestBodies component missing for ref %s", body.Ref)
	}
	target, ok := doc.Components.RequestBodies[name]
	if !ok {
		return nil, fmt.Errorf("requestBody %s not found", name)
	}
	return doc.resolveRequestBody(target, seen, depth+1)
}

func (doc *openAPIDoc) exampleFromMediaType(mt *openAPIMediaType) (any, error) {
	if mt == nil {
		return nil, nil
	}
	if mt.Example != nil {
		return mt.Example, nil
	}
	for _, ex := range mt.Examples {
		if ex.Value != nil {
			return ex.Value, nil
		}
	}
	schema, err := doc.resolveSchema(mt.Schema, map[string]bool{}, 0)
	if err != nil {
		return nil, err
	}
	if schema == nil {
		return nil, nil
	}
	return doc.exampleFromSchema(schema, map[string]bool{}, 0)
}

func (doc *openAPIDoc) resolveSchema(schema *openAPISchema, seen map[string]bool, depth int) (*openAPISchema, error) {
	if schema == nil {
		return nil, nil
	}
	if depth > 32 {
		return nil, errors.New("schema reference depth exceeded")
	}
	if schema.Ref == "" {
		return schema, nil
	}
	name, ok := strings.CutPrefix(schema.Ref, "#/components/schemas/")
	if !ok {
		return nil, fmt.Errorf("unsupported schema ref: %s", schema.Ref)
	}
	if seen[name] {
		return nil, nil
	}
	seen[name] = true
	if doc.Components.Schemas == nil {
		return nil, fmt.Errorf("schemas component missing for ref %s", schema.Ref)
	}
	target, ok := doc.Components.Schemas[name]
	if !ok {
		return nil, fmt.Errorf("schema %s not found", name)
	}
	return doc.resolveSchema(target, seen, depth+1)
}

func (doc *openAPIDoc) exampleFromSchema(schema *openAPISchema, stack map[string]bool, depth int) (any, error) {
	if schema == nil {
		return nil, nil
	}
	if depth > 32 {
		return nil, errors.New("schema recursion limit exceeded")
	}
	if schema.Ref != "" {
		resolved, err := doc.resolveSchema(schema, stack, depth+1)
		if err != nil {
			return nil, err
		}
		return doc.exampleFromSchema(resolved, stack, depth+1)
	}
	if schema.Example != nil {
		return schema.Example, nil
	}
	if schema.Default != nil {
		return schema.Default, nil
	}
	if len(schema.Enum) > 0 {
		return schema.Enum[0], nil
	}
	if len(schema.AllOf) > 0 {
		result := map[string]any{}
		for _, sub := range schema.AllOf {
			val, err := doc.exampleFromSchema(sub, stack, depth+1)
			if err != nil {
				return nil, err
			}
			if val == nil {
				continue
			}
			norm := normalizeYAMLValue(val)
			if obj, ok := norm.(map[string]any); ok {
				for k, v := range obj {
					result[k] = v
				}
			}
		}
		if len(result) > 0 {
			return result, nil
		}
	}
	if len(schema.OneOf) > 0 {
		return doc.exampleFromSchema(schema.OneOf[0], stack, depth+1)
	}
	if len(schema.AnyOf) > 0 {
		return doc.exampleFromSchema(schema.AnyOf[0], stack, depth+1)
	}

	switch schema.Type {
	case "object", "":
		result := map[string]any{}
		if len(schema.Properties) == 0 {
			return result, nil
		}
		keys := make([]string, 0, len(schema.Properties))
		for k := range schema.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sub := schema.Properties[k]
			val, err := doc.exampleFromSchema(sub, stack, depth+1)
			if err != nil {
				return nil, err
			}
			if val == nil {
				val = ""
			}
			result[k] = val
		}
		return result, nil
	case "array":
		val, err := doc.exampleFromSchema(schema.Items, stack, depth+1)
		if err != nil {
			return nil, err
		}
		if val == nil {
			val = ""
		}
		return []any{val}, nil
	case "string":
		return stringExampleForFormat(schema.Format), nil
	case "integer", "number":
		return 0, nil
	case "boolean":
		return false, nil
	}
	return nil, nil
}

func normalizeYAMLValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, vv := range v {
			out[k] = normalizeYAMLValue(vv)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]any, len(v))
		for k, vv := range v {
			out[fmt.Sprint(k)] = normalizeYAMLValue(vv)
		}
		return out
	case []any:
		for i := range v {
			v[i] = normalizeYAMLValue(v[i])
		}
		return v
	default:
		return v
	}
}

func stringExampleForFormat(format string) string {
	switch strings.ToLower(format) {
	case "date-time":
		return "2025-01-01T00:00:00Z"
	case "date":
		return "2025-01-01"
	case "uuid":
		return "00000000-0000-0000-0000-000000000000"
	case "email":
		return "user@example.com"
	case "uri":
		return "https://example.com"
	case "hostname":
		return "example.com"
	case "ipv4":
		return "127.0.0.1"
	case "ipv6":
		return "::1"
	default:
		return "example"
	}
}

func fixturePathForKey(baseDir, key string) string {
	name := strings.ReplaceAll(key, ".", "_")
	return filepath.Join(baseDir, name+".json")
}

// fileCopy describes a file that needs to be copied when writing fixtures
type fileCopy struct {
	Src  string
	Dest string
}

// multipartFixtureDoc is the YAML fixture representation for multipart payloads
type multipartFixtureDoc struct {
	Type  string                 `yaml:"type"`
	Parts []multipartFixturePart `yaml:"parts"`
}

// multipartFixturePart describes a single field in multipart payload
type multipartFixturePart struct {
	Name        string `yaml:"name"`
	Value       string `yaml:"value,omitempty"`
	File        string `yaml:"file,omitempty"`
	Filename    string `yaml:"filename,omitempty"`
	ContentType string `yaml:"contentType,omitempty"`
}

func parseCurlCommand(cmd string) (curlSpec, error) {
	tokens, err := shellSplit(cmd)
	if err != nil {
		return curlSpec{}, err
	}
	if len(tokens) == 0 {
		return curlSpec{}, errors.New("empty curl command")
	}
	if tokens[0] != "curl" {
		return curlSpec{}, errors.New("command must start with curl")
	}

	spec := curlSpec{Method: "GET", Headers: map[string][]string{}}
	var bodyBuilder strings.Builder

	for i := 1; i < len(tokens); i++ {
		tok := tokens[i]
		switch tok {
		case "-X", "--request":
			if i+1 >= len(tokens) {
				return spec, errors.New("missing argument for --request")
			}
			i++
			spec.Method = strings.ToUpper(tokens[i])
		case "-H", "--header":
			if i+1 >= len(tokens) {
				return spec, errors.New("missing argument for --header")
			}
			i++
			h := tokens[i]
			parts := strings.SplitN(h, ":", 2)
			if len(parts) != 2 {
				return spec, fmt.Errorf("invalid header: %s", h)
			}
			name := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			spec.Headers[name] = append(spec.Headers[name], value)
		case "-F", "--form", "--form-string":
			if i+1 >= len(tokens) {
				return spec, errors.New("missing argument for --form")
			}
			i++
			field := tokens[i]
			name, rest, ok := strings.Cut(field, "=")
			if !ok || strings.TrimSpace(name) == "" {
				return spec, fmt.Errorf("invalid form field: %s", field)
			}
			part := formPart{Name: strings.TrimSpace(name)}
			rest = strings.TrimSpace(rest)
			if tok != "--form-string" && strings.HasPrefix(rest, "@") {
				part.IsFile = true
				rest = strings.TrimPrefix(rest, "@")
				segments := strings.Split(rest, ";")
				if len(segments) == 0 || strings.TrimSpace(segments[0]) == "" {
					return spec, fmt.Errorf("invalid file form field: %s", field)
				}
				pathSeg := trimQuotes(strings.TrimSpace(segments[0]))
				if pathSeg == "" {
					return spec, fmt.Errorf("invalid file form field: %s", field)
				}
				part.FilePath = pathSeg
				for _, opt := range segments[1:] {
					opt = strings.TrimSpace(opt)
					if opt == "" {
						continue
					}
					if key, val, ok := strings.Cut(opt, "="); ok {
						switch strings.ToLower(strings.TrimSpace(key)) {
						case "type":
							part.ContentType = trimQuotes(strings.TrimSpace(val))
						case "filename":
							part.FileName = trimQuotes(strings.TrimSpace(val))
						}
					}
				}
				if part.FileName == "" {
					part.FileName = filepath.Base(part.FilePath)
				}
			} else {
				part.Value = rest
			}
			if part.IsFile && strings.TrimSpace(part.FilePath) == "" {
				return spec, fmt.Errorf("invalid file form field: %s", field)
			}
			spec.FormParts = append(spec.FormParts, part)
			if spec.Method == "GET" {
				spec.Method = "POST"
			}
			continue
		case "-d", "--data", "--data-raw", "--data-binary", "--data-urlencode":
			if i+1 >= len(tokens) {
				return spec, errors.New("missing argument for --data")
			}
			i++
			if bodyBuilder.Len() > 0 {
				bodyBuilder.WriteString("&")
			}
			bodyBuilder.WriteString(tokens[i])
		case "--url":
			if i+1 >= len(tokens) {
				return spec, errors.New("missing argument for --url")
			}
			i++
			spec.URL = tokens[i]
		case "--compressed", "--insecure":
			// ignore
		default:
			if strings.HasPrefix(tok, "http://") || strings.HasPrefix(tok, "https://") {
				spec.URL = tok
			} else if strings.HasPrefix(tok, "-") {
				// unsupported flag
				continue
			} else {
				spec.URL = tok
			}
		}
	}

	if bodyBuilder.Len() > 0 {
		spec.Body = bodyBuilder.String()
		if spec.Method == "GET" {
			spec.Method = "POST"
		}
	}

	if spec.URL == "" {
		return spec, errors.New("unable to determine URL from curl command")
	}

	return spec, nil
}

func buildMultipartFixture(spec curlSpec, fixturePath string) (string, []fileCopy, error) {
	if len(spec.FormParts) == 0 {
		return "", nil, errors.New("multipart fixture requires form parts")
	}

	fixtureDir := filepath.Dir(fixturePath)
	baseName := strings.TrimSuffix(filepath.Base(fixturePath), filepath.Ext(fixturePath))
	if baseName == "" {
		baseName = "multipart"
	}

	doc := multipartFixtureDoc{Type: "multipart"}
	var copies []fileCopy
	for idx, part := range spec.FormParts {
		entry := multipartFixturePart{Name: part.Name}
		if part.IsFile {
			if strings.TrimSpace(part.FilePath) == "" {
				return "", nil, fmt.Errorf("form part %q missing file path", part.Name)
			}
			destName := part.FileName
			if strings.TrimSpace(destName) == "" {
				destName = fmt.Sprintf("file_%d", idx+1)
			}
			destName = filepath.Base(destName)
			relPath := filepath.Join(baseName, destName)
			destAbs := filepath.Join(fixtureDir, relPath)
			copies = append(copies, fileCopy{Src: part.FilePath, Dest: destAbs})
			entry.File = relPath
			if part.ContentType != "" {
				entry.ContentType = part.ContentType
			}
			if part.FileName != "" {
				entry.Filename = part.FileName
			} else {
				entry.Filename = destName
			}
		} else {
			entry.Value = part.Value
		}
		doc.Parts = append(doc.Parts, entry)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return "", nil, err
	}
	if err := enc.Close(); err != nil {
		return "", nil, err
	}

	return buf.String(), copies, nil
}

func buildReplayRequest(spec curlSpec) (*http.Request, error) {
	var body io.Reader
	var contentType string

	if spec.HasMultipart() {
		buf := &bytes.Buffer{}
		writer := multipart.NewWriter(buf)
		for _, part := range spec.FormParts {
			if part.IsFile {
				filename := part.FileName
				if strings.TrimSpace(filename) == "" {
					filename = filepath.Base(part.FilePath)
				}
				file, err := os.Open(part.FilePath)
				if err != nil {
					return nil, fmt.Errorf("open form file %s: %w", part.FilePath, err)
				}
				headers := textproto.MIMEHeader{}
				headers.Set("Content-Disposition", fmt.Sprintf("form-data; name=\"%s\"; filename=\"%s\"", escapeQuotes(part.Name), escapeQuotes(filename)))
				if part.ContentType != "" {
					headers.Set("Content-Type", part.ContentType)
				}
				w, err := writer.CreatePart(headers)
				if err != nil {
					file.Close()
					return nil, err
				}
				if _, err := io.Copy(w, file); err != nil {
					file.Close()
					return nil, err
				}
				file.Close()
			} else {
				if err := writer.WriteField(part.Name, part.Value); err != nil {
					return nil, err
				}
			}
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
		contentType = writer.FormDataContentType()
		body = buf
	} else {
		body = strings.NewReader(spec.Body)
	}

	req, err := http.NewRequest(spec.Method, spec.URL, body)
	if err != nil {
		return nil, err
	}
	for k, vv := range spec.Headers {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

func buildAPIEntry(key string, spec curlSpec) loader.Endpoint {
	parsed, err := url.Parse(spec.URL)
	path := spec.URL
	if err == nil {
		path = parsed.Path
		if parsed.RawQuery != "" {
			path += "?" + parsed.RawQuery
		}
	}
	return loader.Endpoint{
		Method:  spec.Method,
		Path:    path,
		Headers: flattenHeaders(spec.Headers),
	}
}

func flattenHeaders(h map[string][]string) map[string]string {
	if len(h) == 0 {
		return nil
	}
	out := map[string]string{}
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out[k] = strings.Join(h[k], ", ")
	}
	return out
}

func defaultFixturePath(apiKey string) string {
	name := strings.ReplaceAll(apiKey, ".", "_")
	return filepath.Join("gherkio", "fixtures", name+".json")
}

func defaultMultipartFixturePath(apiKey string) string {
	name := strings.ReplaceAll(apiKey, ".", "_")
	return filepath.Join("gherkio", "fixtures", name+".multipart.yaml")
}

func defaultFeaturePath(apiKey string) string {
	name := strings.ReplaceAll(apiKey, ".", "_")
	return filepath.Join("gherkio", "features", "imported", name+".feature")
}

func humanizeAPIKey(key string) string {
	parts := strings.Split(key, ".")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, " ")
}

func buildFeatureContent(featureTitle, scenarioTitle, apiKey, fixturePath string, hasBody bool, assertions []string) string {
	var sb strings.Builder
	sb.WriteString("Feature: ")
	sb.WriteString(strings.TrimSpace(featureTitle))
	sb.WriteString("\n\n  Scenario: ")
	sb.WriteString(strings.TrimSpace(scenarioTitle))
	sb.WriteString("\n")
	if hasBody {
		sb.WriteString("    When I call API \"")
		sb.WriteString(apiKey)
		sb.WriteString("\" using fixture \"")
		sb.WriteString(fixturePath)
		sb.WriteString("\"\n")
	} else {
		sb.WriteString("    When I call API \"")
		sb.WriteString(apiKey)
		sb.WriteString("\"\n")
	}
	for _, assertion := range assertions {
		sb.WriteString("    ")
		sb.WriteString(strings.TrimSpace(assertion))
		sb.WriteString("\n")
	}
	return sb.String()
}

func trimFixtureReference(path string) string {
	if path == "" {
		return path
	}
	path = strings.TrimPrefix(path, "./")
	if strings.HasPrefix(path, "gherkio/fixtures/") {
		return strings.TrimPrefix(path, "gherkio/fixtures/")
	}
	return path
}

func prettyJSON(s string) (string, bool) {
	if strings.TrimSpace(s) == "" {
		return s, false
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(s), "", "  "); err != nil {
		return s, false
	}
	return buf.String(), true
}

func writeCatalogEntry(path, key string, endpoint loader.Endpoint) error {
	return writeCatalogEntries(path, map[string]loader.Endpoint{key: endpoint})
}

type catalogFile struct {
	Version   int                           `yaml:"version,omitempty"`
	Auth      map[string]loader.AuthProfile `yaml:"auth,omitempty"`
	Endpoints map[string]loader.Endpoint    `yaml:"endpoints"`
}

func writeCatalogEntries(path string, entries map[string]loader.Endpoint) error {
	if len(entries) == 0 {
		return nil
	}
	cat := catalogFile{Endpoints: map[string]loader.Endpoint{}}
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, &cat); err != nil {
			return fmt.Errorf("parse catalog: %w", err)
		}
	}
	if cat.Endpoints == nil {
		cat.Endpoints = map[string]loader.Endpoint{}
	}
	for k, v := range entries {
		cat.Endpoints[k] = v
	}

	keys := make([]string, 0, len(cat.Endpoints))
	for k := range cat.Endpoints {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := catalogFile{Version: cat.Version, Auth: cat.Auth, Endpoints: map[string]loader.Endpoint{}}
	for _, k := range keys {
		ordered.Endpoints[k] = cat.Endpoints[k]
	}

	data, err := yaml.Marshal(ordered)
	if err != nil {
		return err
	}
	return writeFileEnsureDir(path, string(data))
}

func writeFileEnsureDir(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func copyFileEnsureDir(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func previewText(body []byte) string {
	if len(body) == 0 {
		return " (empty body)"
	}
	max := 200
	preview := string(body)
	if len(preview) > max {
		preview = preview[:max] + "..."
	}
	return "\n" + preview
}

func promptYesNo(msg string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(msg)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(strings.ToLower(text))
	return text == "y" || text == "yes"
}

func buildJSONAssertions(body []byte) []string {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var assertions []string
	for _, k := range keys {
		v := obj[k]
		switch val := v.(type) {
		case string:
			assertions = append(assertions, fmt.Sprintf("And json '$.%s' should equal '%s'", k, escapeSingleQuotes(val)))
		case float64:
			assertions = append(assertions, fmt.Sprintf("And json '$.%s' should be == %v", k, val))
		case bool:
			assertions = append(assertions, fmt.Sprintf("And json '$.%s' should equal '%t'", k, val))
		default:
			// skip arrays/objects/null at top level for MVP
		}
	}
	return assertions
}

func escapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		a := s[0]
		b := s[len(s)-1]
		if (a == '"' && b == '"') || (a == '\'' && b == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func shellSplit(cmd string) ([]string, error) {
	var out []string
	var buf strings.Builder
	inQuote := rune(0)
	escape := false
	for _, r := range cmd {
		if escape {
			buf.WriteRune(r)
			escape = false
			continue
		}
		if r == '\\' {
			escape = true
			continue
		}
		if inQuote != 0 {
			if r == inQuote {
				inQuote = 0
				continue
			}
			buf.WriteRune(r)
			continue
		}
		switch r {
		case '\'', '"':
			inQuote = r
		case ' ', '\t', '\n':
			if buf.Len() > 0 {
				out = append(out, buf.String())
				buf.Reset()
			}
		default:
			buf.WriteRune(r)
		}
	}
	if escape {
		return nil, errors.New("unfinished escape in command")
	}
	if inQuote != 0 {
		return nil, errors.New("unterminated quote in command")
	}
	if buf.Len() > 0 {
		out = append(out, buf.String())
	}
	return out, nil
}