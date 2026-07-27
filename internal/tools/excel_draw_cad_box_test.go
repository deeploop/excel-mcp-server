package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
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

// TestHandleDrawCadBoxParsesRawJSONArguments calls the actual MCP entry
// point (handleDrawCadBox) with arguments shaped like a real decoded
// JSON-RPC tools/call request. See the equivalent hardware-icon test for why
// this matters: a zog schema type mismatch between the "lineStyle" schema
// (StringLike[BorderStyle]) and its destination struct field only panics
// here, not when calling drawCadBox directly.
func TestHandleDrawCadBoxParsesRawJSONArguments(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "cad_box.xlsx")
	if err := excelize.NewFile().SaveAs(filePath); err != nil {
		t.Fatalf("failed to create fixture workbook: %v", err)
	}

	request := mcp.CallToolRequest{}
	request.Params.Name = "excel_draw_cad_box"
	request.Params.Arguments = map[string]any{
		"fileAbsolutePath": filePath,
		"sheetName":        "Sheet1",
		"range":            "B2:F20",
		"leaves":           float64(2),
		"label":            "DR-100",
		"lineColor":        "#000000",
		"lineStyle":        "continuous",
	}

	result, err := handleDrawCadBox(context.Background(), request)
	if err != nil {
		t.Fatalf("handleDrawCadBox failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error result: %s", resultText(t, result))
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
