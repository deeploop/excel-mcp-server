package tools

import (
	"context"
	"fmt"
	"math"

	z "github.com/Oudwins/zog"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/negokaz/excel-mcp-server/internal/excel"
	imcp "github.com/negokaz/excel-mcp-server/internal/mcp"
	"github.com/xuri/excelize/v2"
)

type ExcelDrawCadBoxArguments struct {
	FileAbsolutePath string `zog:"fileAbsolutePath"`
	SheetName        string `zog:"sheetName"`
	Range            string `zog:"range"`
	Leaves           int    `zog:"leaves"`
	Label            string `zog:"label"`
	LineColor        string `zog:"lineColor"`
	LineStyle        string `zog:"lineStyle"`
}

var excelDrawCadBoxArgumentsSchema = z.Struct(z.Shape{
	"fileAbsolutePath": z.String().Test(AbsolutePathTest()).Required(),
	"sheetName":        z.String().Required(),
	"range":            z.String().Required(),
	"leaves":           z.Int().GTE(1).LTE(10).Default(1),
	"label":            z.String(),
	"lineColor":        z.String().Match(colorPattern).Default("#000000"),
	"lineStyle":        z.StringLike[excel.BorderStyle]().OneOf(excel.BorderStyleValues()).Default(excel.BorderStyleContinuous),
})

func AddExcelDrawCadBoxTool(server *server.MCPServer) {
	server.AddTool(mcp.NewTool("excel_draw_cad_box",
		mcp.WithDescription("Draw a CAD-style bordered box (e.g. a door or window frame outline) over a cell range, optionally split into "+
			"multiple leaves by vertical divider lines, then automatically re-read the saved file to verify the borders were actually written correctly."),
		mcp.WithString("fileAbsolutePath",
			mcp.Required(),
			mcp.Description("Absolute path to the Excel file"),
		),
		mcp.WithString("sheetName",
			mcp.Required(),
			mcp.Description("Sheet name where the box is drawn"),
		),
		mcp.WithString("range",
			mcp.Required(),
			mcp.Description("Range of cells the box outline spans (e.g., \"B2:F20\"). Must span more than one row and column."),
		),
		mcp.WithNumber("leaves",
			mcp.Description("Number of leaves (panels) to divide the box into with vertical divider lines (e.g. 2 for a double door). [default: 1]"),
		),
		mcp.WithString("label",
			mcp.Description("Optional text (e.g. a model number) placed in the center cell of the box"),
		),
		mcp.WithString("lineColor",
			mcp.Description("Hex color of the box outline and divider lines (e.g. \"#000000\"). [default: \"#000000\"]"),
		),
		mcp.WithString("lineStyle",
			mcp.Description("Border line style of the box outline and divider lines. [default: \"continuous\"]"),
			mcp.Enum(toStringSlice(excel.BorderStyleValues())...),
		),
	), handleDrawCadBox)
}

func handleDrawCadBox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := ExcelDrawCadBoxArguments{}
	if issues := excelDrawCadBoxArgumentsSchema.Parse(request.Params.Arguments, &args); len(issues) != 0 {
		return imcp.NewToolResultZogIssueMap(issues), nil
	}
	return drawCadBox(args.FileAbsolutePath, args.SheetName, args.Range, args.Leaves, args.Label, args.LineColor, excel.BorderStyle(args.LineStyle))
}

func drawCadBox(fileAbsolutePath string, sheetName string, rangeStr string, leaves int, label string, lineColor string, lineStyle excel.BorderStyle) (*mcp.CallToolResult, error) {
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

	startCol, startRow, endCol, endRow, err := excel.ParseRange(rangeStr)
	if err != nil {
		return imcp.NewToolResultInvalidArgumentError(err.Error()), nil
	}
	if startCol == endCol || startRow == endRow {
		return imcp.NewToolResultInvalidArgumentError("range must span more than one row and column"), nil
	}
	if leaves > endCol-startCol+1 {
		return imcp.NewToolResultInvalidArgumentError("leaves cannot exceed the number of columns in the range"), nil
	}

	dividerCols := computeLeafDividerColumns(startCol, endCol, leaves)

	for row := startRow; row <= endRow; row++ {
		for col := startCol; col <= endCol; col++ {
			var borders []excel.Border
			if col == startCol {
				borders = append(borders, excel.Border{Type: excel.BorderTypeLeft, Style: lineStyle, Color: lineColor})
			}
			if col == endCol {
				borders = append(borders, excel.Border{Type: excel.BorderTypeRight, Style: lineStyle, Color: lineColor})
			}
			if row == startRow {
				borders = append(borders, excel.Border{Type: excel.BorderTypeTop, Style: lineStyle, Color: lineColor})
			}
			if row == endRow {
				borders = append(borders, excel.Border{Type: excel.BorderTypeBottom, Style: lineStyle, Color: lineColor})
			}
			if containsInt(dividerCols, col) {
				borders = append(borders, excel.Border{Type: excel.BorderTypeRight, Style: lineStyle, Color: lineColor})
			}
			if len(borders) == 0 {
				continue
			}
			cellName, err := excelize.CoordinatesToCellName(col, row)
			if err != nil {
				return nil, err
			}
			if err := worksheet.SetCellStyle(cellName, &excel.CellStyle{Border: borders}); err != nil {
				return nil, fmt.Errorf("failed to set border for cell %s: %w", cellName, err)
			}
		}
	}

	if label != "" {
		labelCell, err := excelize.CoordinatesToCellName((startCol+endCol)/2, (startRow+endRow)/2)
		if err != nil {
			return nil, err
		}
		if err := worksheet.SetValue(labelCell, label); err != nil {
			return nil, fmt.Errorf("failed to set label: %w", err)
		}
	}

	if err := workbook.Save(); err != nil {
		return nil, err
	}

	// Automatic verification: re-open the saved file and independently
	// re-read the outer/divider cells' styles and the label value back,
	// rather than trusting that SetCellStyle/Save succeeding means the box
	// was actually drawn correctly.
	verification, err := verifyCadBox(fileAbsolutePath, sheetName, startCol, startRow, endCol, endRow, dividerCols, label)
	if err != nil {
		return nil, fmt.Errorf("failed to automatically verify the drawn box: %w", err)
	}

	text := "# Notice\n"
	text += fmt.Sprintf("backend: %s\n", workbook.GetBackendName())
	text += fmt.Sprintf("CAD box drawn on %s!%s with %d leaf(s).\n", sheetName, rangeStr, leaves)
	text += "# Automatic verification\n"
	if verification.Verified {
		text += "✅ VERIFIED: box outline, divider lines and label all match what was requested.\n"
	} else {
		text += "❌ NOT VERIFIED:\n"
		for _, problem := range verification.Problems {
			text += fmt.Sprintf("- %s\n", problem)
		}
		text += "Do not report this box as successfully drawn without investigating further.\n"
	}
	return mcp.NewToolResultText(text), nil
}

// computeLeafDividerColumns returns the columns (within [startCol, endCol))
// after which a vertical divider line should be drawn to split the range
// into the requested number of leaves.
func computeLeafDividerColumns(startCol, endCol, leaves int) []int {
	if leaves <= 1 {
		return nil
	}
	width := endCol - startCol + 1
	dividers := make([]int, 0, leaves-1)
	for i := 1; i < leaves; i++ {
		boundary := startCol + int(math.Round(float64(width)*float64(i)/float64(leaves))) - 1
		if boundary < startCol {
			boundary = startCol
		}
		if boundary >= endCol {
			boundary = endCol - 1
		}
		dividers = append(dividers, boundary)
	}
	return dividers
}

func containsInt(values []int, target int) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

// CadBoxVerification is the outcome of independently re-reading a drawn CAD
// box's cell styles back from the saved file.
type CadBoxVerification struct {
	Verified bool
	Problems []string
}

func verifyCadBox(fileAbsolutePath string, sheetName string, startCol, startRow, endCol, endRow int, dividerCols []int, label string) (*CadBoxVerification, error) {
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
	checkBorder := func(col, row int, borderType excel.BorderType) {
		cell, err := excelize.CoordinatesToCellName(col, row)
		if err != nil {
			problems = append(problems, err.Error())
			return
		}
		style, err := worksheet.GetCellStyle(cell)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", cell, err))
			return
		}
		found := false
		if style != nil {
			for _, b := range style.Border {
				if b.Type == borderType {
					found = true
					break
				}
			}
		}
		if !found {
			problems = append(problems, fmt.Sprintf("%s is missing its %s border", cell, borderType))
		}
	}

	checkBorder(startCol, startRow, excel.BorderTypeTop)
	checkBorder(startCol, startRow, excel.BorderTypeLeft)
	checkBorder(endCol, startRow, excel.BorderTypeTop)
	checkBorder(endCol, startRow, excel.BorderTypeRight)
	checkBorder(startCol, endRow, excel.BorderTypeBottom)
	checkBorder(startCol, endRow, excel.BorderTypeLeft)
	checkBorder(endCol, endRow, excel.BorderTypeBottom)
	checkBorder(endCol, endRow, excel.BorderTypeRight)

	midRow := (startRow + endRow) / 2
	for _, col := range dividerCols {
		checkBorder(col, midRow, excel.BorderTypeRight)
	}

	if label != "" {
		labelCell, err := excelize.CoordinatesToCellName((startCol+endCol)/2, (startRow+endRow)/2)
		if err != nil {
			return nil, err
		}
		value, err := worksheet.GetValue(labelCell)
		if err != nil || value != label {
			problems = append(problems, fmt.Sprintf("label cell %s expected %q but found %q", labelCell, label, value))
		}
	}

	return &CadBoxVerification{Verified: len(problems) == 0, Problems: problems}, nil
}
