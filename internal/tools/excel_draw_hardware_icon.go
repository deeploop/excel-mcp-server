package tools

import (
	"context"
	"fmt"

	z "github.com/Oudwins/zog"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	excel "github.com/negokaz/excel-mcp-server/internal/excel"
	imcp "github.com/negokaz/excel-mcp-server/internal/mcp"
)

type ExcelDrawHardwareIconArguments struct {
	FileAbsolutePath string  `zog:"fileAbsolutePath"`
	SheetName        string  `zog:"sheetName"`
	Cell             string  `zog:"cell"`
	IconType         string  `zog:"iconType"`
	SizePoints       float64 `zog:"sizePoints"`
}

var excelDrawHardwareIconArgumentsSchema = z.Struct(z.Shape{
	"fileAbsolutePath": z.String().Test(AbsolutePathTest()).Required(),
	"sheetName":        z.String().Required(),
	"cell":             z.String().Required(),
	"iconType":         z.StringLike[excel.HardwareIconType]().OneOf(excel.HardwareIconTypeValues()).Required(),
	"sizePoints":       z.Float64().GT(0).LTE(1000).Default(36),
})

func AddExcelDrawHardwareIconTool(server *server.MCPServer) {
	server.AddTool(mcp.NewTool("excel_draw_hardware_icon",
		mcp.WithDescription("Draw a predefined door/window hardware icon (hinge, handle, lock, etc.) onto an Excel sheet as vector shapes, "+
			"then automatically re-read the saved file to verify the icon was actually written correctly."),
		mcp.WithString("fileAbsolutePath",
			mcp.Required(),
			mcp.Description("Absolute path to the Excel file"),
		),
		mcp.WithString("sheetName",
			mcp.Required(),
			mcp.Description("Sheet name where the icon is drawn"),
		),
		mcp.WithString("cell",
			mcp.Required(),
			mcp.Description("Anchor cell for the icon's top-left corner (e.g. \"C5\")"),
		),
		mcp.WithString("iconType",
			mcp.Required(),
			mcp.Description("Type of hardware icon to draw"),
			mcp.Enum(toStringSlice(excel.HardwareIconTypeValues())...),
		),
		mcp.WithNumber("sizePoints",
			mcp.Description("Width/height of the icon's bounding square, in points. [default: 36]"),
		),
	), handleDrawHardwareIcon)
}

func handleDrawHardwareIcon(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := ExcelDrawHardwareIconArguments{}
	if issues := excelDrawHardwareIconArgumentsSchema.Parse(request.Params.Arguments, &args); len(issues) != 0 {
		return imcp.NewToolResultZogIssueMap(issues), nil
	}
	return drawHardwareIcon(args.FileAbsolutePath, args.SheetName, args.Cell, excel.HardwareIconType(args.IconType), args.SizePoints)
}

func drawHardwareIcon(fileAbsolutePath string, sheetName string, cell string, iconType excel.HardwareIconType, sizePoints float64) (*mcp.CallToolResult, error) {
	workbook, release, err := excel.OpenFile(fileAbsolutePath)
	if err != nil {
		return nil, err
	}
	defer release()

	worksheet, err := workbook.FindSheet(sheetName)
	if err != nil {
		return imcp.NewToolResultInvalidArgumentError(err.Error()), nil
	}
	defer worksheet.Release()

	result, err := worksheet.DrawHardwareIcon(cell, iconType, excel.HardwareIconOptions{SizePoints: sizePoints})
	if err != nil {
		return imcp.NewToolResultInvalidArgumentError(err.Error()), nil
	}
	result.SheetName = sheetName

	if err := workbook.Save(); err != nil {
		return nil, err
	}

	// Automatic verification: don't just trust that DrawHardwareIcon/Save
	// returned without error. Re-open the file that was actually written to
	// disk and independently confirm the icon's shapes are really there.
	verification, err := excel.VerifyHardwareIcon(fileAbsolutePath, result)
	if err != nil {
		return nil, fmt.Errorf("failed to automatically verify the drawn icon: %w", err)
	}

	text := "# Notice\n"
	text += fmt.Sprintf("backend: %s\n", workbook.GetBackendName())
	text += fmt.Sprintf("Hardware icon [%s] drawn at %s!%s.\n", iconType, sheetName, cell)
	text += "# Automatic verification\n"
	if verification.Verified {
		text += fmt.Sprintf("✅ VERIFIED: %s\n", verification.Message)
	} else {
		text += fmt.Sprintf("❌ NOT VERIFIED: %s\n", verification.Message)
		text += "The file was saved, but re-reading it back did not find the expected shapes. " +
			"Do not report this icon as successfully drawn without investigating further.\n"
	}
	return mcp.NewToolResultText(text), nil
}

func toStringSlice[T ~string](values []T) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = string(v)
	}
	return result
}
