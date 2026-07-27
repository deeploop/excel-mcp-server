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

func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatalf("result has no content")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}
	return text.Text
}

func TestDrawHardwareIconToolVerifiesItsOwnOutput(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "hardware_icon.xlsx")
	if err := excelize.NewFile().SaveAs(filePath); err != nil {
		t.Fatalf("failed to create fixture workbook: %v", err)
	}

	result, err := drawHardwareIcon(filePath, "Sheet1", "C5", excel.HardwareIconHinge, 36)
	if err != nil {
		t.Fatalf("drawHardwareIcon failed: %v", err)
	}

	text := resultText(t, result)
	if !strings.Contains(text, "VERIFIED") {
		t.Fatalf("expected result to mention verification status, got: %s", text)
	}
	if strings.Contains(text, "NOT VERIFIED") {
		t.Fatalf("expected the freshly drawn icon to verify successfully, got: %s", text)
	}
}

// TestHandleDrawHardwareIconParsesRawJSONArguments calls the actual MCP
// entry point (handleDrawHardwareIcon), not the internal drawHardwareIcon
// helper, with arguments shaped exactly like what mcp-go hands the handler
// after decoding a real JSON-RPC tools/call request (a map[string]any with
// plain string/float64 values). This is a regression test for a bug where
// the zog schema for "iconType" was declared as StringLike[HardwareIconType]
// but the destination struct field was a plain `string`: that mismatch is
// invisible to Go's compiler and to tests that call drawHardwareIcon
// directly, and only panics inside zog's schema.Parse at request time -
// which previously crashed the whole server process on a real tool call.
func TestHandleDrawHardwareIconParsesRawJSONArguments(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "hardware_icon.xlsx")
	if err := excelize.NewFile().SaveAs(filePath); err != nil {
		t.Fatalf("failed to create fixture workbook: %v", err)
	}

	request := mcp.CallToolRequest{}
	request.Params.Name = "excel_draw_hardware_icon"
	request.Params.Arguments = map[string]any{
		"fileAbsolutePath": filePath,
		"sheetName":        "Sheet1",
		"cell":             "C5",
		"iconType":         "hinge",
	}

	result, err := handleDrawHardwareIcon(context.Background(), request)
	if err != nil {
		t.Fatalf("handleDrawHardwareIcon failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error result: %s", resultText(t, result))
	}
}

func TestDrawHardwareIconToolRejectsUnknownSheet(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "hardware_icon.xlsx")
	if err := excelize.NewFile().SaveAs(filePath); err != nil {
		t.Fatalf("failed to create fixture workbook: %v", err)
	}

	result, err := drawHardwareIcon(filePath, "NoSuchSheet", "C5", excel.HardwareIconHinge, 36)
	if err != nil {
		t.Fatalf("expected a tool-error result, not a Go error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected an error result for an unknown sheet")
	}
}
