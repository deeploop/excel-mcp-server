package tools

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/negokaz/excel-mcp-server/internal/excel"
	"github.com/xuri/excelize/v2"
)

func TestAddMeasurementTableToolVerifiesItsOwnOutput(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "measurement_table.xlsx")
	if err := excelize.NewFile().SaveAs(filePath); err != nil {
		t.Fatalf("failed to create fixture workbook: %v", err)
	}

	extraFields := []MeasurementField{{Label: "Thickness (mm)", Value: 45}}
	result, err := addMeasurementTable(filePath, "Sheet1", "H2", "Door Dimensions", 900, 2100, extraFields)
	if err != nil {
		t.Fatalf("addMeasurementTable failed: %v", err)
	}

	text := resultText(t, result)
	if !strings.Contains(text, "VERIFIED") {
		t.Fatalf("expected result to mention verification status, got: %s", text)
	}
	if strings.Contains(text, "NOT VERIFIED") {
		t.Fatalf("expected the freshly written table to verify successfully, got: %s", text)
	}

	// Spot-check the actual cell values landed where expected:
	// H2=title, H3/I3=headers, H4/I4=width row, H5/I5=height row, H6/I6=extra field row.
	workbook, release, err := excel.OpenFile(filePath)
	if err != nil {
		t.Fatalf("failed to reopen file: %v", err)
	}
	defer release()
	worksheet, err := workbook.FindSheet("Sheet1")
	if err != nil {
		t.Fatalf("FindSheet failed: %v", err)
	}
	defer worksheet.Release()

	if v, _ := worksheet.GetValue("H4"); v != "Width (mm)" {
		t.Errorf("expected H4 to be 'Width (mm)', got %q", v)
	}
	if v, _ := worksheet.GetValue("I4"); v != "900" {
		t.Errorf("expected I4 to be '900', got %q", v)
	}
	if v, _ := worksheet.GetValue("H6"); v != "Thickness (mm)" {
		t.Errorf("expected H6 to be 'Thickness (mm)', got %q", v)
	}
}

func TestAddMeasurementTableToolRejectsUnknownSheet(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "measurement_table.xlsx")
	if err := excelize.NewFile().SaveAs(filePath); err != nil {
		t.Fatalf("failed to create fixture workbook: %v", err)
	}

	result, err := addMeasurementTable(filePath, "NoSuchSheet", "H2", "Dimensions", 900, 2100, nil)
	if err != nil {
		t.Fatalf("expected a tool-error result, not a Go error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected an error result for an unknown sheet")
	}
}
