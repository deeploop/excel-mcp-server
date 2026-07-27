package tools

import (
	"context"
	"fmt"

	z "github.com/Oudwins/zog"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/negokaz/excel-mcp-server/internal/excel"
	imcp "github.com/negokaz/excel-mcp-server/internal/mcp"
)

type ExcelValidateTemplateArguments struct {
	FileAbsolutePath string   `zog:"fileAbsolutePath"`
	RequiredSheets   []string `zog:"requiredSheets"`
}

var excelValidateTemplateArgumentsSchema = z.Struct(z.Shape{
	"fileAbsolutePath": z.String().Test(AbsolutePathTest()).Required(),
	"requiredSheets":   z.Slice(z.String()).Required(),
})

func AddExcelValidateTemplateTool(server *server.MCPServer) {
	server.AddTool(mcp.NewTool("excel_validate_template",
		mcp.WithDescription("Verify that a local .xlsx template has every required sheet, and that each one's OOXML relationship chain "+
			"actually resolves - catching tab-index or relationship corruption before it breaks a later drawing/write operation."),
		mcp.WithString("fileAbsolutePath",
			mcp.Required(),
			mcp.Description("Absolute path to the Excel file"),
		),
		mcp.WithArray("requiredSheets",
			mcp.Required(),
			mcp.Description("Sheet names that must exist and resolve correctly"),
			mcp.Items(map[string]any{"type": "string"}),
		),
	), handleValidateTemplate)
}

func handleValidateTemplate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := ExcelValidateTemplateArguments{}
	if issues := excelValidateTemplateArgumentsSchema.Parse(request.Params.Arguments, &args); len(issues) != 0 {
		return imcp.NewToolResultZogIssueMap(issues), nil
	}
	results, err := excel.ValidateTemplate(args.FileAbsolutePath, args.RequiredSheets)
	if err != nil {
		return imcp.NewToolResultInvalidArgumentError(err.Error()), nil
	}

	allOK := true
	text := "# Template validation\n"
	for _, r := range results {
		if r.OK {
			text += fmt.Sprintf("✅ %s\n", r.SheetName)
		} else {
			allOK = false
			text += fmt.Sprintf("❌ %s: %s\n", r.SheetName, r.Error)
		}
	}
	result := mcp.NewToolResultText(text)
	result.IsError = !allOK
	return result, nil
}
