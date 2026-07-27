package tools

import (
	"context"
	"fmt"
	"math"
	"strconv"

	z "github.com/Oudwins/zog"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/negokaz/excel-mcp-server/internal/excel"
	imcp "github.com/negokaz/excel-mcp-server/internal/mcp"
	"github.com/xuri/excelize/v2"
)

type MeasurementField struct {
	Label string  `zog:"label"`
	Value float64 `zog:"value"`
}

type ExcelAddMeasurementTableArguments struct {
	FileAbsolutePath string             `zog:"fileAbsolutePath"`
	SheetName        string             `zog:"sheetName"`
	Cell             string             `zog:"cell"`
	Title            string             `zog:"title"`
	WidthMm          float64            `zog:"widthMm"`
	HeightMm         float64            `zog:"heightMm"`
	ExtraFields      []MeasurementField `zog:"extraFields"`
}

var excelAddMeasurementTableArgumentsSchema = z.Struct(z.Shape{
	"fileAbsolutePath": z.String().Test(AbsolutePathTest()).Required(),
	"sheetName":        z.String().Required(),
	"cell":             z.String().Required(),
	"title":            z.String().Default("Dimensions"),
	"widthMm":          z.Float64().GT(0).Required(),
	"heightMm":         z.Float64().GT(0).Required(),
	"extraFields": z.Slice(z.Struct(z.Shape{
		"label": z.String().Required(),
		"value": z.Float64().Required(),
	})).Default([]MeasurementField{}),
})

func AddExcelAddMeasurementTableTool(server *server.MCPServer) {
	server.AddTool(mcp.NewTool("excel_add_measurement_table",
		mcp.WithDescription("Put a height/width measurement table on the Excel sheet (e.g. door or window dimensions), "+
			"then automatically re-read the saved file to verify every value was actually written correctly."),
		mcp.WithString("fileAbsolutePath",
			mcp.Required(),
			mcp.Description("Absolute path to the Excel file"),
		),
		mcp.WithString("sheetName",
			mcp.Required(),
			mcp.Description("Sheet name where the table is placed"),
		),
		mcp.WithString("cell",
			mcp.Required(),
			mcp.Description("Top-left anchor cell of the table (e.g. \"H2\"). The table occupies two columns starting here."),
		),
		mcp.WithString("title",
			mcp.Description("Title text placed above the table. [default: \"Dimensions\"]"),
		),
		mcp.WithNumber("widthMm",
			mcp.Required(),
			mcp.Description("Width measurement, in millimeters"),
		),
		mcp.WithNumber("heightMm",
			mcp.Required(),
			mcp.Description("Height measurement, in millimeters"),
		),
		mcp.WithArray("extraFields",
			mcp.Description("Additional measurement rows (e.g. thickness, leaf count) appended below width/height"),
			mcp.Items(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"label": map[string]any{"type": "string"},
					"value": map[string]any{"type": "number"},
				},
				"required": []string{"label", "value"},
			}),
		),
	), handleAddMeasurementTable)
}

func handleAddMeasurementTable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := ExcelAddMeasurementTableArguments{}
	if issues := excelAddMeasurementTableArgumentsSchema.Parse(request.Params.Arguments, &args); len(issues) != 0 {
		return imcp.NewToolResultZogIssueMap(issues), nil
	}
	return addMeasurementTable(args.FileAbsolutePath, args.SheetName, args.Cell, args.Title, args.WidthMm, args.HeightMm, args.ExtraFields)
}

// measurementTableRows returns the ordered label/value rows shown under the
// table's header row (width and height first, then any extra fields).
func measurementTableRows(widthMm, heightMm float64, extraFields []MeasurementField) []MeasurementField {
	rows := make([]MeasurementField, 0, 2+len(extraFields))
	rows = append(rows, MeasurementField{Label: "Width (mm)", Value: widthMm})
	rows = append(rows, MeasurementField{Label: "Height (mm)", Value: heightMm})
	rows = append(rows, extraFields...)
	return rows
}

func addMeasurementTable(fileAbsolutePath string, sheetName string, cell string, title string, widthMm float64, heightMm float64, extraFields []MeasurementField) (*mcp.CallToolResult, error) {
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

	anchorCol, anchorRow, _, _, err := excel.ParseRange(cell)
	if err != nil {
		return imcp.NewToolResultInvalidArgumentError(err.Error()), nil
	}

	titleCell, err := excelize.CoordinatesToCellName(anchorCol, anchorRow)
	if err != nil {
		return nil, err
	}
	if err := worksheet.SetValue(titleCell, title); err != nil {
		return nil, fmt.Errorf("failed to set title: %w", err)
	}
	bold := true
	if err := worksheet.SetCellStyle(titleCell, &excel.CellStyle{Font: &excel.FontStyle{Bold: &bold}}); err != nil {
		return nil, fmt.Errorf("failed to style title: %w", err)
	}

	headerRow := anchorRow + 1
	labelHeaderCell, err := excelize.CoordinatesToCellName(anchorCol, headerRow)
	if err != nil {
		return nil, err
	}
	valueHeaderCell, err := excelize.CoordinatesToCellName(anchorCol+1, headerRow)
	if err != nil {
		return nil, err
	}
	if err := worksheet.SetValue(labelHeaderCell, "Measurement"); err != nil {
		return nil, fmt.Errorf("failed to set header: %w", err)
	}
	if err := worksheet.SetValue(valueHeaderCell, "Value"); err != nil {
		return nil, fmt.Errorf("failed to set header: %w", err)
	}
	headerStyle := &excel.CellStyle{
		Font: &excel.FontStyle{Bold: &bold},
		Fill: &excel.FillStyle{Type: excel.FillTypePattern, Pattern: excel.FillPatternSolid, Color: []string{"#D9D9D9"}},
	}
	if err := worksheet.SetCellStyle(labelHeaderCell, headerStyle); err != nil {
		return nil, fmt.Errorf("failed to style header: %w", err)
	}
	if err := worksheet.SetCellStyle(valueHeaderCell, headerStyle); err != nil {
		return nil, fmt.Errorf("failed to style header: %w", err)
	}

	rows := measurementTableRows(widthMm, heightMm, extraFields)
	for i, row := range rows {
		rowIndex := headerRow + 1 + i
		labelCell, err := excelize.CoordinatesToCellName(anchorCol, rowIndex)
		if err != nil {
			return nil, err
		}
		valueCell, err := excelize.CoordinatesToCellName(anchorCol+1, rowIndex)
		if err != nil {
			return nil, err
		}
		if err := worksheet.SetValue(labelCell, row.Label); err != nil {
			return nil, fmt.Errorf("failed to set row label: %w", err)
		}
		if err := worksheet.SetValue(valueCell, row.Value); err != nil {
			return nil, fmt.Errorf("failed to set row value: %w", err)
		}
	}

	if err := workbook.Save(); err != nil {
		return nil, err
	}

	// Automatic verification: re-open the saved file and independently
	// re-read the title, header, and every row's label/value back, rather
	// than trusting that SetValue/Save succeeding means the table is
	// actually correct.
	verification, err := verifyMeasurementTable(fileAbsolutePath, sheetName, anchorCol, anchorRow, title, rows)
	if err != nil {
		return nil, fmt.Errorf("failed to automatically verify the measurement table: %w", err)
	}

	text := "# Notice\n"
	text += fmt.Sprintf("backend: %s\n", workbook.GetBackendName())
	text += fmt.Sprintf("Measurement table placed on %s!%s.\n", sheetName, titleCell)
	text += "# Automatic verification\n"
	if verification.Verified {
		text += "✅ VERIFIED: title, header and all measurement rows match what was written.\n"
	} else {
		text += "❌ NOT VERIFIED:\n"
		for _, problem := range verification.Problems {
			text += fmt.Sprintf("- %s\n", problem)
		}
		text += "Do not report this table as successfully written without investigating further.\n"
	}
	return mcp.NewToolResultText(text), nil
}

// MeasurementTableVerification is the outcome of independently re-reading a
// written measurement table's cell values back from the saved file.
type MeasurementTableVerification struct {
	Verified bool
	Problems []string
}

func verifyMeasurementTable(fileAbsolutePath string, sheetName string, anchorCol, anchorRow int, title string, rows []MeasurementField) (*MeasurementTableVerification, error) {
	workbook, release, err := excel.OpenFile(fileAbsolutePath)
	if err != nil {
		return nil, err
	}
	defer release()

	worksheet, err := workbook.FindSheet(sheetName)
	if err != nil {
		return nil, err
	}
	defer worksheet.Release()

	var problems []string
	checkText := func(col, row int, expected string) {
		cell, err := excelize.CoordinatesToCellName(col, row)
		if err != nil {
			problems = append(problems, err.Error())
			return
		}
		actual, err := worksheet.GetValue(cell)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", cell, err))
			return
		}
		if actual != expected {
			problems = append(problems, fmt.Sprintf("%s expected %q but found %q", cell, expected, actual))
		}
	}
	checkNumber := func(col, row int, expected float64) {
		cell, err := excelize.CoordinatesToCellName(col, row)
		if err != nil {
			problems = append(problems, err.Error())
			return
		}
		actual, err := worksheet.GetValue(cell)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", cell, err))
			return
		}
		actualValue, err := strconv.ParseFloat(actual, 64)
		if err != nil || math.Abs(actualValue-expected) > 1e-9 {
			problems = append(problems, fmt.Sprintf("%s expected %v but found %q", cell, expected, actual))
		}
	}

	checkText(anchorCol, anchorRow, title)
	headerRow := anchorRow + 1
	checkText(anchorCol, headerRow, "Measurement")
	checkText(anchorCol+1, headerRow, "Value")

	for i, row := range rows {
		rowIndex := headerRow + 1 + i
		checkText(anchorCol, rowIndex, row.Label)
		checkNumber(anchorCol+1, rowIndex, row.Value)
	}

	return &MeasurementTableVerification{Verified: len(problems) == 0, Problems: problems}, nil
}
