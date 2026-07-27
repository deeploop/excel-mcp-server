package tools

import (
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
