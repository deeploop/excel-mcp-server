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

func newOoxmlFixtureWithIcon(t *testing.T) string {
	t.Helper()
	filePath := filepath.Join(t.TempDir(), "ooxml_fixture.xlsx")
	if err := excelize.NewFile().SaveAs(filePath); err != nil {
		t.Fatalf("failed to create fixture workbook: %v", err)
	}
	// drawHardwareIcon is the actual tool entry point already covered
	// elsewhere; reuse it here just to give the sheet a real drawing part.
	if _, err := drawHardwareIcon(filePath, "Sheet1", "C5", excel.HardwareIconHinge, 36); err != nil {
		t.Fatalf("failed to seed fixture with a drawing: %v", err)
	}
	return filePath
}

// Each of these calls the real handle* MCP entry point with arguments
// shaped like a decoded JSON-RPC tools/call request (map[string]any with
// plain string/float64/[]any values), not the internal Go helper - this is
// the path that previously caught a zog schema panic invisible to
// Go-value-typed tests (see excel_draw_hardware_icon_test.go).

func TestHandleOoxmlResolveSheetParsesRawJSONArguments(t *testing.T) {
	filePath := newOoxmlFixtureWithIcon(t)

	request := mcp.CallToolRequest{}
	request.Params.Name = "excel_ooxml_resolve_sheet"
	request.Params.Arguments = map[string]any{
		"fileAbsolutePath": filePath,
		"sheetName":        "Sheet1",
	}

	result, err := handleOoxmlResolveSheet(context.Background(), request)
	if err != nil {
		t.Fatalf("handleOoxmlResolveSheet failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error result: %s", resultText(t, result))
	}
}

func TestHandleOoxmlInspectRelsParsesRawJSONArguments(t *testing.T) {
	filePath := newOoxmlFixtureWithIcon(t)

	request := mcp.CallToolRequest{}
	request.Params.Name = "excel_ooxml_inspect_rels"
	request.Params.Arguments = map[string]any{
		"fileAbsolutePath": filePath,
		"partPath":         "xl/worksheets/sheet1.xml",
	}

	result, err := handleOoxmlInspectRels(context.Background(), request)
	if err != nil {
		t.Fatalf("handleOoxmlInspectRels failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error result: %s", resultText(t, result))
	}
}

func TestHandleOoxmlExtractDrawingsParsesRawJSONArguments(t *testing.T) {
	filePath := newOoxmlFixtureWithIcon(t)

	request := mcp.CallToolRequest{}
	request.Params.Name = "excel_ooxml_extract_drawings"
	request.Params.Arguments = map[string]any{
		"fileAbsolutePath": filePath,
		"sheetName":        "Sheet1",
	}

	result, err := handleOoxmlExtractDrawings(context.Background(), request)
	if err != nil {
		t.Fatalf("handleOoxmlExtractDrawings failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error result: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "roundRect") {
		t.Fatalf("expected extracted drawing text to mention roundRect, got: %s", text)
	}
}
