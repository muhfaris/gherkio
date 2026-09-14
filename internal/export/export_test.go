package export

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestWriteXLSX(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out", "bulk.xlsx")

	headers := []string{"customer_id", "amount", "status"}
	rows := [][]interface{}{
		{"C1", 100.5, "normal"},
		{"C2", 2000, "priority"},
	}

	if err := WriteXLSX(path, "BulkData", headers, rows); err != nil {
		t.Fatalf("WriteXLSX failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open workbook: %v", err)
	}
	defer f.Close()

	// Verify header row.
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		got, _ := f.GetCellValue("BulkData", cell)
		if got != h {
			t.Errorf("header cell %s = %q, want %q", cell, got, h)
		}
	}

	// Verify data rows.
	checks := map[string]string{
		"A2": "C1",
		"B2": "100.5",
		"C2": "normal",
		"A3": "C2",
		"B3": "2000",
		"C3": "priority",
	}
	for cell, want := range checks {
		got, _ := f.GetCellValue("BulkData", cell)
		if got != want {
			t.Errorf("cell %s = %q, want %q", cell, got, want)
		}
	}
}

func TestWriteXLSXDefaultSheet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "default.xlsx")

	if err := WriteXLSX(path, "", []string{"a"}, [][]interface{}{{"1"}}); err != nil {
		t.Fatalf("WriteXLSX failed: %v", err)
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open workbook: %v", err)
	}
	defer f.Close()

	got, _ := f.GetCellValue("Sheet1", "A1")
	if got != "a" {
		t.Errorf("default sheet header = %q, want %q", got, "a")
	}
}

func TestWriteXLSXLargeDataset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large_export.xlsx")

	headers := []string{
		"id",
		"sku",
		"product_name",
		"quantity",
		"unit_price",
		"total_amount",
		"in_stock",
		"category",
		"discount_pct",
		"created_at",
	}

	const totalRows = 150
	rows := make([][]interface{}, totalRows)
	for i := 0; i < totalRows; i++ {
		id := i + 1
		qty := (i % 20) + 1
		unitPrice := 10.5 + float64(i)*1.25
		totalAmount := float64(qty) * unitPrice
		inStock := (i % 5) != 0
		category := "Standard"
		if unitPrice > 100 {
			category = "Premium"
		}
		discount := 0.05 * float64(i%4)
		createdAt := "2026-09-14T10:00:00Z"

		rows[i] = []interface{}{
			id,
			"SKU-" + string(rune('A'+(i%26))) + "-00" + string(rune('0'+(i%10))),
			"Product Item #" + string(rune('0'+(i%10))),
			qty,
			unitPrice,
			totalAmount,
			inStock,
			category,
			discount,
			createdAt,
		}
	}

	if err := WriteXLSX(path, "Inventory", headers, rows); err != nil {
		t.Fatalf("WriteXLSX failed for large dataset: %v", err)
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open generated workbook: %v", err)
	}
	defer f.Close()

	// Verify all 10 headers
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		got, _ := f.GetCellValue("Inventory", cell)
		if got != h {
			t.Errorf("header col %d (%s) = %q, want %q", i+1, cell, got, h)
		}
	}

	// Verify row count by reading all sheet rows
	sheetRows, err := f.GetRows("Inventory")
	if err != nil {
		t.Fatalf("failed to read rows: %v", err)
	}
	// 1 header row + 150 data rows = 151 rows
	if len(sheetRows) != 151 {
		t.Fatalf("expected 151 rows (including header), got %d", len(sheetRows))
	}
	if len(sheetRows[0]) != 10 {
		t.Fatalf("expected 10 columns, got %d", len(sheetRows[0]))
	}

	// Spot check row 100
	row100 := sheetRows[100] // 100th data row is index 100 (1-based row 101)
	if row100[0] != "100" {
		t.Errorf("row 100 id = %q, want '100'", row100[0])
	}
}
