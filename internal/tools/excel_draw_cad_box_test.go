package tools

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/negokaz/excel-mcp-server/internal/excel"
	"github.com/xuri/excelize/v2"
)

func TestDrawCadBoxToolVerifiesItsOwnOutput(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "cad_box.xlsx")
	if err := excelize.NewFile().SaveAs(filePath); err != nil {
		t.Fatalf("failed to create fixture workbook: %v", err)
	}

	result, err := drawCadBox(filePath, "Sheet1", "B2:F20", 2, "DR-100", "#000000", excel.BorderStyleContinuous)
	if err != nil {
		t.Fatalf("drawCadBox failed: %v", err)
	}

	text := resultText(t, result)
	if !strings.Contains(text, "VERIFIED") {
		t.Fatalf("expected result to mention verification status, got: %s", text)
	}
	if strings.Contains(text, "NOT VERIFIED") {
		t.Fatalf("expected the freshly drawn box to verify successfully, got: %s", text)
	}
}

func TestDrawCadBoxToolRejectsDegenerateRange(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "cad_box.xlsx")
	if err := excelize.NewFile().SaveAs(filePath); err != nil {
		t.Fatalf("failed to create fixture workbook: %v", err)
	}

	result, err := drawCadBox(filePath, "Sheet1", "B2:B2", 1, "", "#000000", excel.BorderStyleContinuous)
	if err != nil {
		t.Fatalf("expected a tool-error result, not a Go error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected an error result for a single-cell range")
	}
}

func TestDrawCadBoxToolRejectsTooManyLeaves(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "cad_box.xlsx")
	if err := excelize.NewFile().SaveAs(filePath); err != nil {
		t.Fatalf("failed to create fixture workbook: %v", err)
	}

	result, err := drawCadBox(filePath, "Sheet1", "B2:C10", 5, "", "#000000", excel.BorderStyleContinuous)
	if err != nil {
		t.Fatalf("expected a tool-error result, not a Go error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected an error result when leaves exceeds the number of columns")
	}
}
