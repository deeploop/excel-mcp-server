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

// --- excel_ooxml_resolve_sheet ---

type ExcelOoxmlResolveSheetArguments struct {
	FileAbsolutePath string `zog:"fileAbsolutePath"`
	SheetName        string `zog:"sheetName"`
}

var excelOoxmlResolveSheetArgumentsSchema = z.Struct(z.Shape{
	"fileAbsolutePath": z.String().Test(AbsolutePathTest()).Required(),
	"sheetName":        z.String().Required(),
})

func AddExcelOoxmlResolveSheetTool(server *server.MCPServer) {
	server.AddTool(mcp.NewTool("excel_ooxml_resolve_sheet",
		mcp.WithDescription("Resolve the exact worksheet part path (e.g. xl/worksheets/sheet1.xml) and its .rels part path for a sheet, "+
			"by walking the real xl/workbook.xml relationship chain in the saved file."),
		mcp.WithString("fileAbsolutePath",
			mcp.Required(),
			mcp.Description("Absolute path to the Excel file"),
		),
		mcp.WithString("sheetName",
			mcp.Required(),
			mcp.Description("Sheet name to resolve"),
		),
	), handleOoxmlResolveSheet)
}

func handleOoxmlResolveSheet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := ExcelOoxmlResolveSheetArguments{}
	if issues := excelOoxmlResolveSheetArgumentsSchema.Parse(request.Params.Arguments, &args); len(issues) != 0 {
		return imcp.NewToolResultZogIssueMap(issues), nil
	}
	resolved, err := excel.ResolveSheetPart(args.FileAbsolutePath, args.SheetName)
	if err != nil {
		return imcp.NewToolResultInvalidArgumentError(err.Error()), nil
	}
	text := fmt.Sprintf("worksheet part: %s\nrelationships part: %s\n", resolved.WorksheetPath, resolved.RelsPath)
	return mcp.NewToolResultText(text), nil
}

// --- excel_ooxml_inspect_rels ---

type ExcelOoxmlInspectRelsArguments struct {
	FileAbsolutePath string `zog:"fileAbsolutePath"`
	PartPath         string `zog:"partPath"`
}

var excelOoxmlInspectRelsArgumentsSchema = z.Struct(z.Shape{
	"fileAbsolutePath": z.String().Test(AbsolutePathTest()).Required(),
	"partPath":         z.String().Required(),
})

func AddExcelOoxmlInspectRelsTool(server *server.MCPServer) {
	server.AddTool(mcp.NewTool("excel_ooxml_inspect_rels",
		mcp.WithDescription("List every relationship (Id, Type, Target) defined for an OOXML part's .rels file, to debug broken template links "+
			"(e.g. a worksheet part whose drawing relationship is missing or points at a deleted part)."),
		mcp.WithString("fileAbsolutePath",
			mcp.Required(),
			mcp.Description("Absolute path to the Excel file"),
		),
		mcp.WithString("partPath",
			mcp.Required(),
			mcp.Description("OOXML part path to inspect, relative to the zip root (e.g. \"xl/worksheets/sheet1.xml\")"),
		),
	), handleOoxmlInspectRels)
}

func handleOoxmlInspectRels(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := ExcelOoxmlInspectRelsArguments{}
	if issues := excelOoxmlInspectRelsArgumentsSchema.Parse(request.Params.Arguments, &args); len(issues) != 0 {
		return imcp.NewToolResultZogIssueMap(issues), nil
	}
	rels, err := excel.InspectRelationships(args.FileAbsolutePath, args.PartPath)
	if err != nil {
		return imcp.NewToolResultInvalidArgumentError(err.Error()), nil
	}
	text := fmt.Sprintf("# Relationships for %s\n", args.PartPath)
	if len(rels) == 0 {
		text += "(none)\n"
	}
	for _, r := range rels {
		text += fmt.Sprintf("- %s -> %s (%s)\n", r.ID, r.Target, r.Type)
	}
	return mcp.NewToolResultText(text), nil
}

// --- excel_ooxml_extract_drawings ---

type ExcelOoxmlExtractDrawingsArguments struct {
	FileAbsolutePath string `zog:"fileAbsolutePath"`
	SheetName        string `zog:"sheetName"`
}

var excelOoxmlExtractDrawingsArgumentsSchema = z.Struct(z.Shape{
	"fileAbsolutePath": z.String().Test(AbsolutePathTest()).Required(),
	"sheetName":        z.String().Required(),
})

func AddExcelOoxmlExtractDrawingsTool(server *server.MCPServer) {
	server.AddTool(mcp.NewTool("excel_ooxml_extract_drawings",
		mcp.WithDescription("Walk the sheet.xml -> sheet.xml.rels -> drawingN.xml relationship chain and list every shape/picture anchored on "+
			"the sheet, with each one's cell-range anchor (e.g. \"B2\" to \"E10\") already resolved to A1 notation."),
		mcp.WithString("fileAbsolutePath",
			mcp.Required(),
			mcp.Description("Absolute path to the Excel file"),
		),
		mcp.WithString("sheetName",
			mcp.Required(),
			mcp.Description("Sheet name to extract drawings from"),
		),
	), handleOoxmlExtractDrawings)
}

func handleOoxmlExtractDrawings(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := ExcelOoxmlExtractDrawingsArguments{}
	if issues := excelOoxmlExtractDrawingsArgumentsSchema.Parse(request.Params.Arguments, &args); len(issues) != 0 {
		return imcp.NewToolResultZogIssueMap(issues), nil
	}
	shapes, err := excel.ExtractDrawingShapes(args.FileAbsolutePath, args.SheetName)
	if err != nil {
		return imcp.NewToolResultInvalidArgumentError(err.Error()), nil
	}
	text := fmt.Sprintf("# Drawings on %s\n", args.SheetName)
	if len(shapes) == 0 {
		text += "(none)\n"
	}
	for _, s := range shapes {
		if s.Kind == "shape" {
			text += fmt.Sprintf("- shape[%s] anchored %s:%s\n", s.PrstGeom, s.FromCell, s.ToCell)
		} else {
			text += fmt.Sprintf("- picture anchored %s:%s\n", s.FromCell, s.ToCell)
		}
	}
	return mcp.NewToolResultText(text), nil
}
