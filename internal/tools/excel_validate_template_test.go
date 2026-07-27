package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/xuri/excelize/v2"
)

func TestHandleValidateTemplateParsesRawJSONArguments(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "validate_template.xlsx")
	if err := excelize.NewFile().SaveAs(filePath); err != nil {
		t.Fatalf("failed to create fixture workbook: %v", err)
	}

	request := mcp.CallToolRequest{}
	request.Params.Name = "excel_validate_template"
	request.Params.Arguments = map[string]any{
		"fileAbsolutePath": filePath,
		"requiredSheets":   []any{"Sheet1", "NoSuchSheet"},
	}

	result, err := handleValidateTemplate(context.Background(), request)
	if err != nil {
		t.Fatalf("handleValidateTemplate failed: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected an error result because NoSuchSheet is missing, got: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "✅ Sheet1") || !strings.Contains(text, "❌ NoSuchSheet") {
		t.Fatalf("expected per-sheet validation results, got: %s", text)
	}
}
