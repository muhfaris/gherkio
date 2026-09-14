package export

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

// WriteXLSX creates an Excel workbook at the given path with a header row and
// data rows. The sheet name defaults to "Sheet1" when empty.
func WriteXLSX(path, sheet string, headers []string, rows [][]interface{}) error {
	if sheet == "" {
		sheet = "Sheet1"
	}

	f := excelize.NewFile()
	defer f.Close()

	// Use the default sheet created by excelize as the target sheet.
	index, err := f.NewSheet(sheet)
	if err != nil {
		return fmt.Errorf("failed to create sheet %q: %w", sheet, err)
	}
	f.SetActiveSheet(index)

	// Write header row.
	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return fmt.Errorf("failed to compute header cell: %w", err)
		}
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return fmt.Errorf("failed to write header cell %s: %w", cell, err)
		}
	}

	// Write data rows.
	for r, row := range rows {
		for c, val := range row {
			cell, err := excelize.CoordinatesToCellName(c+1, r+2)
			if err != nil {
				return fmt.Errorf("failed to compute data cell: %w", err)
			}
			if err := f.SetCellValue(sheet, cell, val); err != nil {
				return fmt.Errorf("failed to write cell %s: %w", cell, err)
			}
		}
	}

	// Ensure the parent directory exists.
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create output directory %q: %w", dir, err)
		}
	}

	if err := f.SaveAs(path); err != nil {
		return fmt.Errorf("failed to save workbook: %w", err)
	}

	return nil
}
