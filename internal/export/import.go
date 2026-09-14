package export

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/muhfaris/gherkio/internal/model"
	"github.com/xuri/excelize/v2"
)

var (
	nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)
	multipleUnderscores  = regexp.MustCompile(`_+`)
)

// NormalizeHeader converts a raw header string (e.g. "Customer Name ($)") into
// a clean, valid snake_case identifier (e.g. "customer_name").
func NormalizeHeader(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	// Replace non-alphanumeric runes with underscore
	var builder strings.Builder
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(unicode.ToLower(r))
		} else {
			builder.WriteRune('_')
		}
	}

	normalized := builder.String()
	// Collapse consecutive underscores
	normalized = multipleUnderscores.ReplaceAllString(normalized, "_")
	// Trim leading and trailing underscores
	normalized = strings.Trim(normalized, "_")

	return normalized
}

// ReadXLSX reads an Excel workbook and parses rows into an array of maps.
func ReadXLSX(path string, sheet string, headerRow int, dataStartRow int, columns []model.ImportColumn) ([]map[string]interface{}, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file %q: %w", path, err)
	}
	defer f.Close()

	// Resolve sheet name
	if sheet == "" {
		sheetList := f.GetSheetList()
		if len(sheetList) == 0 {
			return nil, fmt.Errorf("excel file %q has no sheets", path)
		}
		activeIdx := f.GetActiveSheetIndex()
		if activeIdx >= 0 && activeIdx < len(sheetList) {
			sheet = sheetList[activeIdx]
		} else {
			sheet = sheetList[0]
		}
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to read rows from sheet %q: %w", sheet, err)
	}

	if headerRow <= 0 {
		headerRow = 1
	}
	if dataStartRow <= 0 {
		dataStartRow = headerRow + 1
	}

	if len(rows) < headerRow {
		return []map[string]interface{}{}, nil
	}

	rawHeaders := rows[headerRow-1]

	// Build alias lookup from explicit columns config if provided
	aliasMap := make(map[string]string)
	for _, col := range columns {
		if col.Header != "" && col.As != "" {
			aliasMap[strings.TrimSpace(col.Header)] = col.As
			aliasMap[NormalizeHeader(col.Header)] = col.As
		}
	}

	// Map column index to field key
	headerKeys := make([]string, len(rawHeaders))
	for i, h := range rawHeaders {
		trimmedHeader := strings.TrimSpace(h)
		if alias, ok := aliasMap[trimmedHeader]; ok {
			headerKeys[i] = alias
			continue
		}
		norm := NormalizeHeader(trimmedHeader)
		if alias, ok := aliasMap[norm]; ok {
			headerKeys[i] = alias
			continue
		}
		if norm == "" {
			norm = fmt.Sprintf("column_%d", i+1)
		}
		headerKeys[i] = norm
	}

	var result []map[string]interface{}

	for r := dataStartRow - 1; r < len(rows); r++ {
		row := rows[r]
		// Skip completely empty rows
		isEmpty := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			continue
		}

		rowMap := make(map[string]interface{})
		for c, key := range headerKeys {
			if key == "" {
				continue
			}
			var rawVal string
			if c < len(row) {
				rawVal = strings.TrimSpace(row[c])
			}
			rowMap[key] = parseCellValue(rawVal)
		}
		result = append(result, rowMap)
	}

	if result == nil {
		result = []map[string]interface{}{}
	}

	return result, nil
}

func parseCellValue(val string) interface{} {
	if val == "" {
		return ""
	}
	// Try boolean
	if strings.EqualFold(val, "true") {
		return true
	}
	if strings.EqualFold(val, "false") {
		return false
	}
	// Try integer
	if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
		return intVal
	}
	// Try float
	if floatVal, err := strconv.ParseFloat(val, 64); err == nil && strings.Contains(val, ".") {
		return floatVal
	}
	return val
}
