package excel

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestDrawHardwareIconAndVerify(t *testing.T) {
	for _, iconType := range HardwareIconTypeValues() {
		t.Run(string(iconType), func(t *testing.T) {
			file := excelize.NewFile()
			defer file.Close()
			file.Path = filepath.Join(t.TempDir(), "hardware_icon.xlsx")

			workbook := NewExcelizeExcel(file)
			worksheet, err := workbook.FindSheet("Sheet1")
			if err != nil {
				t.Fatalf("FindSheet failed: %v", err)
			}
			defer worksheet.Release()

			result, err := worksheet.DrawHardwareIcon("C5", iconType, HardwareIconOptions{})
			if err != nil {
				t.Fatalf("DrawHardwareIcon failed: %v", err)
			}
			if result.AnchorCell != "C5" {
				t.Errorf("expected anchor cell C5, got %s", result.AnchorCell)
			}

			expectedPrimitives, err := HardwareIconPrimitives(iconType)
			if err != nil {
				t.Fatalf("HardwareIconPrimitives failed: %v", err)
			}
			if len(result.ExpectedShapeTypes) != len(expectedPrimitives) {
				t.Fatalf("expected %d shapes, got %d", len(expectedPrimitives), len(result.ExpectedShapeTypes))
			}

			if err := workbook.Save(); err != nil {
				t.Fatalf("Save failed: %v", err)
			}

			verification, err := VerifyHardwareIcon(file.Path, result)
			if err != nil {
				t.Fatalf("VerifyHardwareIcon failed: %v", err)
			}
			if !verification.Verified {
				t.Fatalf("expected icon to be verified, got: %+v", verification)
			}
		})
	}
}

func TestVerifyHardwareIconDetectsMissingShapes(t *testing.T) {
	file := excelize.NewFile()
	defer file.Close()
	file.Path = filepath.Join(t.TempDir(), "hardware_icon_missing.xlsx")

	workbook := NewExcelizeExcel(file)
	worksheet, err := workbook.FindSheet("Sheet1")
	if err != nil {
		t.Fatalf("FindSheet failed: %v", err)
	}
	defer worksheet.Release()

	if err := workbook.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Nothing was drawn, so verifying a claimed result against the saved
	// file must fail rather than report a false positive.
	fakeResult := &HardwareIconResult{
		SheetName:          "Sheet1",
		AnchorCell:         "C5",
		IconType:           HardwareIconHinge,
		ExpectedShapeTypes: []string{"roundRect", "roundRect", "ellipse"},
	}
	verification, err := VerifyHardwareIcon(file.Path, fakeResult)
	if err != nil {
		t.Fatalf("VerifyHardwareIcon failed: %v", err)
	}
	if verification.Verified {
		t.Fatalf("expected verification to fail when nothing was drawn")
	}
}

func TestDrawHardwareIconUnknownType(t *testing.T) {
	file := excelize.NewFile()
	defer file.Close()

	workbook := NewExcelizeExcel(file)
	worksheet, err := workbook.FindSheet("Sheet1")
	if err != nil {
		t.Fatalf("FindSheet failed: %v", err)
	}
	defer worksheet.Release()

	if _, err := worksheet.DrawHardwareIcon("C5", HardwareIconType("not-a-real-icon"), HardwareIconOptions{}); err == nil {
		t.Fatal("expected an error for an unknown icon type")
	}
}
