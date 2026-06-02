package runner

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/muhfaris/gherkio/internal/model"
)

func TestProjectCollection_BasicAndFilter(t *testing.T) {
	// Sample data
	rawQuestions := []interface{}{
		map[string]interface{}{
			"question_template_id": 1001.0,
			"input_type":           "text",
			"answer_type":          "string",
			"seq":                  1.0,
			"free_text_answer":     "My answer",
		},
		map[string]interface{}{
			"question_template_id": 1002.0,
			"input_type":           "radiobutton",
			"answer_type":          "option",
			"seq":                  2.0,
			"free_text_answer":     "",
		},
	}

	vars := map[string]interface{}{
		"raw_questions": rawQuestions,
		"survey_id":     132.0,
	}

	cfg := &model.ProjectionConfig{
		From: "$raw_questions",
		As:   "question",
		Where: map[string]interface{}{
			"$question.input_type": "radiobutton",
		},
		Select: map[string]interface{}{
			"question_template_id": "$question.question_template_id",
			"input_type":           "$question.input_type",
			"answer_type":          "$question.answer_type",
			"seq":                  "$question.seq",
			"is_answered":          true,
			"survey_id_ref":        "$survey_id",
		},
	}

	got, err := ProjectCollection(cfg, vars)
	if err != nil {
		t.Fatalf("ProjectCollection failed: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(got))
	}

	item := got[0].(map[string]interface{})
	if item["question_template_id"] != 1002.0 {
		t.Errorf("Expected question_template_id to be 1002.0, got %v", item["question_template_id"])
	}
	if item["is_answered"] != true {
		t.Errorf("Expected is_answered to be true, got %v", item["is_answered"])
	}
	if item["survey_id_ref"] != 132.0 {
		t.Errorf("Expected survey_id_ref to be 132.0, got %v", item["survey_id_ref"])
	}
}

func TestProjectCollection_MatcherFilteringAndLimit(t *testing.T) {
	rawCartItems := []interface{}{
		map[string]interface{}{"id": 1.0, "quantity": 5.0, "in_stock": true},
		map[string]interface{}{"id": 2.0, "quantity": 0.0, "in_stock": true},
		map[string]interface{}{"id": 3.0, "quantity": 10.0, "in_stock": false},
		map[string]interface{}{"id": 4.0, "quantity": 2.0, "in_stock": true},
	}

	vars := map[string]interface{}{
		"cart_items": rawCartItems,
	}

	cfg := &model.ProjectionConfig{
		From: "$cart_items",
		As:   "item",
		Where: map[string]interface{}{
			"item.in_stock": true,
			"item.quantity": "$gt(0)", // normalizes to "gt 0"
		},
		Limit: 1,
		Select: map[string]interface{}{
			"productId": "$item.id",
			"qty":       "$item.quantity",
		},
	}

	got, err := ProjectCollection(cfg, vars)
	if err != nil {
		t.Fatalf("ProjectCollection failed: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("Expected limit to restrict results to 1, got %d", len(got))
	}

	item := got[0].(map[string]interface{})
	if item["productId"] != 1.0 {
		t.Errorf("Expected first matching item (id: 1.0), got %v", item["productId"])
	}
}

func TestProjectCollection_NestedProjectionsAndTypePreservation(t *testing.T) {
	// Nested array in parent item
	surveyData := []interface{}{
		map[string]interface{}{
			"id":   101.0,
			"name": "Survey 1",
			"options": []interface{}{
				map[string]interface{}{"opt_id": 1.0, "text": "Yes"},
				map[string]interface{}{"opt_id": 2.0, "text": "No"},
			},
		},
	}

	vars := map[string]interface{}{
		"surveys": surveyData,
	}

	// We'll define a nested projection as map[string]interface{} inside Select
	cfg := &model.ProjectionConfig{
		From: "$surveys",
		As:   "survey",
		Select: map[string]interface{}{
			"surveyId": "$survey.id",
			"title":    "Name: $survey.name",
			"choices": map[string]interface{}{
				"from": "$survey.options",
				"as":   "opt",
				"select": map[string]interface{}{
					"choiceId": "$opt.opt_id",
					"label":    "$opt.text",
				},
			},
		},
	}

	got, err := ProjectCollection(cfg, vars)
	if err != nil {
		t.Fatalf("ProjectCollection failed: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("Expected 1 survey, got %d", len(got))
	}

	survey := got[0].(map[string]interface{})
	if survey["surveyId"] != 101.0 {
		t.Errorf("Expected surveyId to be preserved as 101.0 float64, got %v (%T)", survey["surveyId"], survey["surveyId"])
	}
	if survey["title"] != "Name: Survey 1" {
		t.Errorf("Expected string interpolation, got %v", survey["title"])
	}

	choices := survey["choices"].([]interface{})
	if len(choices) != 2 {
		t.Fatalf("Expected 2 sub-projection choices, got %d", len(choices))
	}

	choice1 := choices[0].(map[string]interface{})
	if choice1["choiceId"] != 1.0 {
		t.Errorf("Expected choiceId to be 1.0, got %v", choice1["choiceId"])
	}
	if choice1["label"] != "Yes" {
		t.Errorf("Expected label to be 'Yes', got %v", choice1["label"])
	}
}

func TestProjectCollection_MissingPropertyJSONNull(t *testing.T) {
	products := []interface{}{
		map[string]interface{}{
			"id":   55.0,
			"meta": map[string]interface{}{
				// "tag" is missing
			},
		},
	}

	vars := map[string]interface{}{
		"products": products,
	}

	cfg := &model.ProjectionConfig{
		From: "$products",
		As:   "p",
		Select: map[string]interface{}{
			"id":  "$p.id",
			"tag": "$p.meta.tag", // nested missing property
		},
	}

	got, err := ProjectCollection(cfg, vars)
	if err != nil {
		t.Fatalf("ProjectCollection failed: %v", err)
	}

	item := got[0].(map[string]interface{})
	if item["id"] != 55.0 {
		t.Errorf("Expected id to be 55.0, got %v", item["id"])
	}

	// In Go JSON maps, nil represents null
	if val, ok := item["tag"]; !ok || val != nil {
		t.Errorf("Expected missing tag to evaluate as nil (null), got ok=%v, val=%v", ok, val)
	}
}

func TestProjectCollection_MissingSourceGraceful(t *testing.T) {
	// from refers to non-existent variable
	cfg := &model.ProjectionConfig{
		From:   "$non_existent",
		Select: map[string]interface{}{"id": "$item.id"},
	}

	got, err := ProjectCollection(cfg, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Expected graceful handling, got error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Expected empty list, got %v", got)
	}
}

func TestProjectCollection_NonArraySourceError(t *testing.T) {
	cfg := &model.ProjectionConfig{
		From:   "$not_an_array",
		Select: map[string]interface{}{"id": "$item.id"},
	}

	vars := map[string]interface{}{
		"not_an_array": "just a string",
	}

	_, err := ProjectCollection(cfg, vars)
	if err == nil {
		t.Fatal("Expected error for non-array source, got nil")
	}
}

func TestInterpolateRequest_Integration(t *testing.T) {
	req := model.Request{
		Method: "POST",
		URL:    "http://api.example.com/submit",
		Body: map[string]interface{}{
			"session_id": 999.0,
		},
		Transform: map[string]*model.ProjectionConfig{
			"survey.answers": {
				From: "$questions",
				As:   "q",
				Select: map[string]interface{}{
					"q_id": "$q.id",
					"ans":  "$q.answer",
				},
			},
		},
	}

	vars := map[string]interface{}{
		"questions": []interface{}{
			map[string]interface{}{"id": 101.0, "answer": "Opt A"},
			map[string]interface{}{"id": 102.0, "answer": "Opt B"},
		},
	}

	interpolated, err := InterpolateRequest(req, vars)
	if err != nil {
		t.Fatalf("InterpolateRequest failed: %v", err)
	}

	bodyMap, ok := interpolated.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected interpolated body to be map, got %T", interpolated.Body)
	}

	if bodyMap["session_id"] != 999.0 {
		t.Errorf("Expected session_id to be preserved, got %v", bodyMap["session_id"])
	}

	surveyVal, ok := bodyMap["survey"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected survey key in body, got %v", bodyMap["survey"])
	}

	answersList, ok := surveyVal["answers"].([]interface{})
	if !ok {
		t.Fatalf("Expected answers to be a slice, got %T", surveyVal["answers"])
	}

	if len(answersList) != 2 {
		t.Fatalf("Expected 2 answers, got %d", len(answersList))
	}

	ans1 := answersList[0].(map[string]interface{})
	if ans1["q_id"] != 101.0 {
		t.Errorf("Expected q_id of first item to be 101.0, got %v", ans1["q_id"])
	}
	if ans1["ans"] != "Opt A" {
		t.Errorf("Expected ans of first item to be 'Opt A', got %v", ans1["ans"])
	}
}

func TestWritePath(t *testing.T) {
	data := map[string]interface{}{
		"a": map[string]interface{}{
			"b": 1.0,
		},
	}

	writePath(data, "a.b", 2.0)
	val, ok := resolvePath(data, "a.b")
	if !ok || val != 2.0 {
		t.Errorf("Expected a.b to be updated to 2.0, got %v", val)
	}

	writePath(data, "a.c.d", "new")
	val, ok = resolvePath(data, "a.c.d")
	if !ok || val != "new" {
		t.Errorf("Expected nested maps to be created and a.c.d set to 'new', got %v", val)
	}

	// Overwriting blocking type
	writePath(data, "a.b.c", "overwritten")
	val, ok = resolvePath(data, "a.b.c")
	if !ok || val != "overwritten" {
		t.Errorf("Expected blocking float64 at a.b to be overwritten by a map and a.b.c set, got %v", val)
	}
}

func TestNormalizeMatcherString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"$gt(0)", "gt 0"},
		{"$gte(10)", "gte 10"},
		{"$lt(5)", "lt 5"},
		{"$lte(20)", "lte 20"},
		{"gt 5", "gt 5"},
		{"$uuid", "$uuid"}, // no parentheses, so ignore
	}

	for _, tt := range tests {
		got := normalizeMatcherString(tt.input)
		if got != tt.want {
			t.Errorf("normalizeMatcherString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGetRemainingPath(t *testing.T) {
	tests := []struct {
		key   string
		alias string
		want  string
	}{
		{"$question.type", "question", "type"},
		{"question.type", "question", "type"},
		{"$question", "question", ""},
		{"question", "question", ""},
		{"type", "question", "type"},
		{"$type", "question", "type"},
	}

	for _, tt := range tests {
		got := getRemainingPath(tt.key, tt.alias)
		if got != tt.want {
			t.Errorf("getRemainingPath(%q, %q) = %q, want %q", tt.key, tt.alias, got, tt.want)
		}
	}
}

func TestMapToProjectionConfig(t *testing.T) {
	m := map[string]interface{}{
		"from": "$items",
		"as":   "item",
		"where": map[string]interface{}{
			"item.id": 1.0,
		},
		"limit": 5.0,
		"select": map[string]interface{}{
			"id": "$item.id",
		},
	}

	cfg, ok := mapToProjectionConfig(m)
	if !ok {
		t.Fatal("mapToProjectionConfig failed")
	}

	if cfg.From != "$items" {
		t.Errorf("Expected From to be '$items', got %q", cfg.From)
	}
	if cfg.As != "item" {
		t.Errorf("Expected As to be 'item', got %q", cfg.As)
	}
	if cfg.Limit != 5 {
		t.Errorf("Expected Limit to be 5, got %d", cfg.Limit)
	}
	if _, ok := cfg.Where["item.id"]; !ok {
		t.Errorf("Expected Where to contain 'item.id', got %v", cfg.Where)
	}
	if cfg.Select["id"] != "$item.id" {
		t.Errorf("Expected Select to contain 'id': '$item.id', got %v", cfg.Select)
	}
}

func TestMapToProjectionConfig_Invalid(t *testing.T) {
	// Missing from/select keys
	m := map[string]interface{}{
		"from": "$items",
	}
	_, ok := mapToProjectionConfig(m)
	if ok {
		t.Fatal("Expected mapToProjectionConfig to return false for incomplete map")
	}
}

func TestEvaluateSelectValue_StaticNestedMapAndSlice(t *testing.T) {
	localVars := map[string]interface{}{
		"item": map[string]interface{}{
			"id":   42.0,
			"tags": []interface{}{"tech", "sale"},
		},
	}

	// We pass a select containing a static map, a static slice, and primitive resolution
	selectVal := map[string]interface{}{
		"productId": "$item.id",
		"meta": map[string]interface{}{
			"tags": "$item.tags",
			"flag": true,
		},
		"simpleList": []interface{}{
			"Prefix: $item.id",
			100.0,
		},
	}

	got, err := evaluateSelectValue(selectVal, localVars)
	if err != nil {
		t.Fatalf("evaluateSelectValue failed: %v", err)
	}

	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", got)
	}

	if m["productId"] != 42.0 {
		t.Errorf("Expected 42.0, got %v", m["productId"])
	}

	meta := m["meta"].(map[string]interface{})
	tags := meta["tags"].([]interface{})
	if len(tags) != 2 || tags[0] != "tech" {
		t.Errorf("Expected tags slice, got %v", tags)
	}
	if meta["flag"] != true {
		t.Errorf("Expected true, got %v", meta["flag"])
	}

	list := m["simpleList"].([]interface{})
	if list[0] != "Prefix: 42" {
		t.Errorf("Expected interpolated first list item, got %v", list[0])
	}
	if list[1] != 100.0 {
		t.Errorf("Expected static second list item, got %v", list[1])
	}
}

func TestResolveTypePreserving_MissingParentAlias(t *testing.T) {
	// If the parent alias is completely missing from localVars (not even initialized),
	// we should fall back to interpolateString, which returns error.
	localVars := map[string]interface{}{}

	_, err := resolveTypePreserving("$non_existent.id", localVars)
	if err == nil {
		t.Fatal("Expected error for non-existent parent alias variable lookup, got nil")
	}
}

// Test Gherkio JSON reflection schema schema compatibility
func TestSchemaCompatibility(t *testing.T) {
	// Ensures that the ProjectionConfig can be correctly marshaled to JSON
	cfg := &model.ProjectionConfig{
		From:  "$items",
		As:    "item",
		Limit: 2,
		Select: map[string]interface{}{
			"id": "$item.id",
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled model.ProjectionConfig
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !reflect.DeepEqual(cfg, &unmarshaled) {
		t.Errorf("deep equal check failed\n  cfg: %+v\n  got: %+v", cfg, &unmarshaled)
	}
}

func TestProjectCollection_TypeCasting(t *testing.T) {
	rawItems := []interface{}{
		map[string]interface{}{
			"id":         1001.0,
			"str_id":     "42",
			"float_id":   "12.34",
			"is_active":  "true",
			"active_num": 1.0,
			"null_val":   nil,
		},
	}

	vars := map[string]interface{}{
		"items": rawItems,
	}

	cfg := &model.ProjectionConfig{
		From: "$items",
		As:   "item",
		Select: map[string]interface{}{
			"id_as_string":    "$string(item.id)",
			"id_as_string_2":  "$string($item.id)", // testing with $ prefix
			"str_id_as_int":   "$int(item.str_id)",
			"float_id_as_f":   "$float(item.float_id)",
			"is_active_as_b":  "$bool(item.is_active)",
			"active_num_as_b": "$bool(item.active_num)",
			"null_as_string":  "$string(item.null_val)",
			"missing_as_str":  "$string(item.missing_key)", // missing key inside parent alias -> returns null
		},
	}

	got, err := ProjectCollection(cfg, vars)
	if err != nil {
		t.Fatalf("ProjectCollection failed: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(got))
	}

	item := got[0].(map[string]interface{})

	if item["id_as_string"] != "1001" {
		t.Errorf("Expected '1001', got %v (%T)", item["id_as_string"], item["id_as_string"])
	}
	if item["id_as_string_2"] != "1001" {
		t.Errorf("Expected '1001' with $ prefix, got %v (%T)", item["id_as_string_2"], item["id_as_string_2"])
	}
	if item["str_id_as_int"] != 42 {
		t.Errorf("Expected 42, got %v (%T)", item["str_id_as_int"], item["str_id_as_int"])
	}
	if item["float_id_as_f"] != 12.34 {
		t.Errorf("Expected 12.34, got %v (%T)", item["float_id_as_f"], item["float_id_as_f"])
	}
	if item["is_active_as_b"] != true {
		t.Errorf("Expected true, got %v (%T)", item["is_active_as_b"], item["is_active_as_b"])
	}
	if item["active_num_as_b"] != true {
		t.Errorf("Expected true, got %v (%T)", item["active_num_as_b"], item["active_num_as_b"])
	}
	if item["null_as_string"] != nil {
		t.Errorf("Expected nil for null cast, got %v", item["null_as_string"])
	}
	if item["missing_as_str"] != nil {
		t.Errorf("Expected nil for missing property, got %v", item["missing_as_str"])
	}
}

func TestInterpolateRequest_BodyCasting(t *testing.T) {
	req := model.Request{
		Method: "POST",
		URL:    "http://api.example.com/submit",
		Body: map[string]interface{}{
			"id_as_int":       "$employee_id",
			"id_as_string":    "$string(employee_id)",
			"id_as_string_2":  "$string($employee_id)",
			"status_as_bool":  "$bool(status_flag)",
			"price_as_float":  "$float(price_str)",
			"missing_val":     "$string(missing_val)",
		},
	}

	vars := map[string]interface{}{
		"employee_id": 1234,
		"status_flag": "true",
		"price_str":   "99.99",
		"missing_val": nil,
	}

	interpolated, err := InterpolateRequest(req, vars)
	if err != nil {
		t.Fatalf("InterpolateRequest failed: %v", err)
	}

	bodyMap, ok := interpolated.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected body to be map, got %T", interpolated.Body)
	}

	if bodyMap["id_as_int"] != 1234 {
		t.Errorf("Expected integer 1234, got %v (%T)", bodyMap["id_as_int"], bodyMap["id_as_int"])
	}
	if bodyMap["id_as_string"] != "1234" {
		t.Errorf("Expected string '1234', got %v (%T)", bodyMap["id_as_string"], bodyMap["id_as_string"])
	}
	if bodyMap["id_as_string_2"] != "1234" {
		t.Errorf("Expected string '1234', got %v (%T)", bodyMap["id_as_string_2"], bodyMap["id_as_string_2"])
	}
	if bodyMap["status_as_bool"] != true {
		t.Errorf("Expected boolean true, got %v (%T)", bodyMap["status_as_bool"], bodyMap["status_as_bool"])
	}
	if bodyMap["price_as_float"] != 99.99 {
		t.Errorf("Expected float 99.99, got %v (%T)", bodyMap["price_as_float"], bodyMap["price_as_float"])
	}
	if bodyMap["missing_val"] != nil {
		t.Errorf("Expected nil for missing_val, got %v (%T)", bodyMap["missing_val"], bodyMap["missing_val"])
	}

	// Test strict error behavior for completely undefined variables
	errReq := model.Request{
		Method: "POST",
		URL:    "http://api.example.com/submit",
		Body: map[string]interface{}{
			"invalid_cast": "$string(non_existent_key)",
		},
	}
	_, err = InterpolateRequest(errReq, vars)
	if err == nil {
		t.Error("Expected error for casting completely undefined variable, but got none")
	}
}

func TestProjectCollection_ConditionalIf(t *testing.T) {
	items := []interface{}{
		map[string]interface{}{
			"id":               1.0,
			"type":             "radiobutton",
			"is_answered":      false,
			"selected_option":  "opt_a",
			"free_text_answer": "",
		},
		map[string]interface{}{
			"id":               2.0,
			"type":             "text",
			"is_answered":      true,
			"selected_option":  "",
			"free_text_answer": "My feedback",
		},
	}

	vars := map[string]interface{}{
		"items": items,
	}

	t.Run("truthy condition uses then value", func(t *testing.T) {
		cfg := &model.ProjectionConfig{
			From: "$items",
			As:   "item",
			Select: map[string]interface{}{
				"id": "$item.id",
				"answer": "$if(item.is_answered, $item.free_text_answer, $item.selected_option)",
			},
		}

		got, err := ProjectCollection(cfg, vars)
		if err != nil {
			t.Fatalf("ProjectCollection failed: %v", err)
		}

		if len(got) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(got))
		}

		// Item 0: is_answered=false -> should use else value (selected_option)
		item0 := got[0].(map[string]interface{})
		if item0["id"] != 1.0 {
			t.Errorf("Expected id 1.0, got %v", item0["id"])
		}
		if item0["answer"] != "opt_a" {
			t.Errorf("Expected answer 'opt_a' (default/else), got %v", item0["answer"])
		}

		// Item 1: is_answered=true -> should use then value (free_text_answer)
		item1 := got[1].(map[string]interface{})
		if item1["id"] != 2.0 {
			t.Errorf("Expected id 2.0, got %v", item1["id"])
		}
		if item1["answer"] != "My feedback" {
			t.Errorf("Expected answer 'My feedback' (then), got %v", item1["answer"])
		}
	})

	t.Run("missing condition falls to else clause", func(t *testing.T) {
		itemsWithMissing := []interface{}{
			map[string]interface{}{
				"id":      3.0,
				"default": "fallback_val",
			},
		}

		varsMissing := map[string]interface{}{
			"items": itemsWithMissing,
		}

		cfg := &model.ProjectionConfig{
			From: "$items",
			As:   "item",
			Select: map[string]interface{}{
				"id":     "$item.id",
				"result": "$if(item.missing_field, $item.custom_val, $item.default)",
			},
		}

		got, err := ProjectCollection(cfg, varsMissing)
		if err != nil {
			t.Fatalf("ProjectCollection failed: %v", err)
		}

		if len(got) != 1 {
			t.Fatalf("Expected 1 item, got %d", len(got))
		}

		item := got[0].(map[string]interface{})
		if item["result"] != "fallback_val" {
			t.Errorf("Expected 'fallback_val' from else clause, got %v", item["result"])
		}
	})

	t.Run("no else clause returns null for falsy condition", func(t *testing.T) {
		cfg := &model.ProjectionConfig{
			From: "$items",
			As:   "item",
			Select: map[string]interface{}{
				"id":   "$item.id",
				"only": "$if(item.is_answered, $item.free_text_answer)",
			},
		}

		got, err := ProjectCollection(cfg, vars)
		if err != nil {
			t.Fatalf("ProjectCollection failed: %v", err)
		}

		// Item 0: is_answered=false, no else -> null
		item0 := got[0].(map[string]interface{})
		if item0["only"] != nil {
			t.Errorf("Expected nil for falsy condition with no else, got %v", item0["only"])
		}

		// Item 1: is_answered=true -> uses then value
		item1 := got[1].(map[string]interface{})
		if item1["only"] != "My feedback" {
			t.Errorf("Expected 'My feedback' for truthy condition, got %v", item1["only"])
		}
	})

	t.Run("then value can use type casting", func(t *testing.T) {
		cfg := &model.ProjectionConfig{
			From: "$items",
			As:   "item",
			Select: map[string]interface{}{
				"id_as_str": "$if(item.is_answered, $string(item.free_text_answer), $string(item.selected_option))",
			},
		}

		got, err := ProjectCollection(cfg, vars)
		if err != nil {
			t.Fatalf("ProjectCollection failed: %v", err)
		}

		// Item 1: is_answered=true -> $string(item.free_text_answer) -> "My feedback"
		item1 := got[1].(map[string]interface{})
		if item1["id_as_str"] != "My feedback" {
			t.Errorf("Expected 'My feedback' (string), got %v (%T)", item1["id_as_str"], item1["id_as_str"])
		}
	})

	t.Run("nil condition value is falsy", func(t *testing.T) {
		nilItems := []interface{}{
			map[string]interface{}{
				"id":        4.0,
				"nil_field": nil,
				"backup":    "backup_val",
			},
		}

		varsNil := map[string]interface{}{
			"items": nilItems,
		}

		cfg := &model.ProjectionConfig{
			From: "$items",
			As:   "item",
			Select: map[string]interface{}{
				"id":  "$item.id",
				"val": "$if(item.nil_field, $item.should_not_appear, $item.backup)",
			},
		}

		got, err := ProjectCollection(cfg, varsNil)
		if err != nil {
			t.Fatalf("ProjectCollection failed: %v", err)
		}

		item := got[0].(map[string]interface{})
		if item["val"] != "backup_val" {
			t.Errorf("Expected 'backup_val' since nil condition is falsy, got %v", item["val"])
		}
	})

	t.Run("invalid $if syntax returns error", func(t *testing.T) {
		_, err := resolveTypePreserving("$if(only_one_arg)", vars)
		if err == nil {
			t.Error("Expected error for $if with single argument, got nil")
		}
	})
}

func TestInterpolateRequest_BodyConditionalIf(t *testing.T) {
	vars := map[string]interface{}{
		"is_active":  true,
		"active_id":  42,
		"fallback":   "fb_val",
		"is_empty":   false,
		"empty_val":  "should_not_appear",
		"empty_fb":   "fallback_for_empty",
	}

	t.Run("truthy condition uses then value in body", func(t *testing.T) {
		req := model.Request{
			Method: "POST",
			URL:    "http://api.example.com/test",
			Body: map[string]interface{}{
				"value": "$if(is_active, $active_id, $fallback)",
			},
		}

		got, err := InterpolateRequest(req, vars)
		if err != nil {
			t.Fatalf("InterpolateRequest failed: %v", err)
		}

		bodyMap, ok := got.Body.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map body, got %T", got.Body)
		}
		if bodyMap["value"] != 42 {
			t.Errorf("Expected 42 (then), got %v", bodyMap["value"])
		}
	})

	t.Run("falsy condition uses else value in body", func(t *testing.T) {
		req := model.Request{
			Method: "POST",
			URL:    "http://api.example.com/test",
			Body: map[string]interface{}{
				"value": "$if(is_empty, $empty_val, $empty_fb)",
			},
		}

		got, err := InterpolateRequest(req, vars)
		if err != nil {
			t.Fatalf("InterpolateRequest failed: %v", err)
		}

		bodyMap, ok := got.Body.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map body, got %T", got.Body)
		}
		if bodyMap["value"] != "fallback_for_empty" {
			t.Errorf("Expected 'fallback_for_empty' (else), got %v", bodyMap["value"])
		}
	})
}

