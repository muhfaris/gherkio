package export

import (
	"path/filepath"
	"testing"

	"github.com/muhfaris/gherkio/internal/model"
)

func TestNormalizeHeader(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Customer Name", "customer_name"},
		{"Order ID ($)", "order_id"},
		{"  Total Amount (USD)  ", "total_amount_usd"},
		{"product_sku_code", "product_sku_code"},
		{"---Special---Header---", "special_header"},
		{"123 Number Field", "123_number_field"},
		{"", ""},
	}

	for _, tt := range tests {
		got := NormalizeHeader(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeHeader(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestReadXLSX(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_import.xlsx")

	headers := []string{"Customer ID", "Full Name", "Balance", "Active"}
	rows := [][]interface{}{
		{"CUST-001", "Alice Smith", 125.50, true},
		{"CUST-002", "Bob Jones", 0.00, false},
		{"CUST-003", "Charlie Brown", 500, true},
	}

	if err := WriteXLSX(path, "Customers", headers, rows); err != nil {
		t.Fatalf("WriteXLSX failed: %v", err)
	}

	// 1. Test auto-normalized headers
	data, err := ReadXLSX(path, "Customers", 1, 2, nil)
	if err != nil {
		t.Fatalf("ReadXLSX failed: %v", err)
	}

	if len(data) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(data))
	}

	if data[0]["customer_id"] != "CUST-001" {
		t.Errorf("row 0 customer_id = %v, want CUST-001", data[0]["customer_id"])
	}
	if data[0]["full_name"] != "Alice Smith" {
		t.Errorf("row 0 full_name = %v, want Alice Smith", data[0]["full_name"])
	}
	if data[0]["active"] != true {
		t.Errorf("row 0 active = %v, want true", data[0]["active"])
	}

	// 2. Test explicit column aliases
	columnMappings := []model.ImportColumn{
		{Header: "Customer ID", As: "id"},
		{Header: "Full Name", As: "name"},
		{Header: "Balance", As: "amount"},
	}

	dataAliased, err := ReadXLSX(path, "Customers", 1, 2, columnMappings)
	if err != nil {
		t.Fatalf("ReadXLSX with aliases failed: %v", err)
	}

	if dataAliased[1]["id"] != "CUST-002" {
		t.Errorf("row 1 id = %v, want CUST-002", dataAliased[1]["id"])
	}
	if dataAliased[1]["name"] != "Bob Jones" {
		t.Errorf("row 1 name = %v, want Bob Jones", dataAliased[1]["name"])
	}
	if dataAliased[1]["active"] != false {
		t.Errorf("row 1 active = %v, want false", dataAliased[1]["active"])
	}
}
