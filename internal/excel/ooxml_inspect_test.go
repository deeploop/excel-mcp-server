package excel

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func newFixtureWithHardwareIcon(t *testing.T) string {
	t.Helper()
	file := excelize.NewFile()
	defer file.Close()
	file.Path = filepath.Join(t.TempDir(), "ooxml_inspect.xlsx")

	workbook := NewExcelizeExcel(file)
	worksheet, err := workbook.FindSheet("Sheet1")
	if err != nil {
		t.Fatalf("FindSheet failed: %v", err)
	}
	defer worksheet.Release()

	if _, err := worksheet.DrawHardwareIcon("C5", HardwareIconHinge, HardwareIconOptions{}); err != nil {
		t.Fatalf("DrawHardwareIcon failed: %v", err)
	}
	if err := workbook.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	return file.Path
}

func TestResolveSheetPart(t *testing.T) {
	filePath := newFixtureWithHardwareIcon(t)

	resolved, err := ResolveSheetPart(filePath, "Sheet1")
	if err != nil {
		t.Fatalf("ResolveSheetPart failed: %v", err)
	}
	if resolved.WorksheetPath != "xl/worksheets/sheet1.xml" {
		t.Errorf("unexpected worksheet path: %s", resolved.WorksheetPath)
	}
	if resolved.RelsPath != "xl/worksheets/_rels/sheet1.xml.rels" {
		t.Errorf("unexpected rels path: %s", resolved.RelsPath)
	}

	if _, err := ResolveSheetPart(filePath, "NoSuchSheet"); err == nil {
		t.Fatal("expected an error for an unknown sheet")
	}
}

func TestInspectRelationships(t *testing.T) {
	filePath := newFixtureWithHardwareIcon(t)

	rels, err := InspectRelationships(filePath, "xl/worksheets/sheet1.xml")
	if err != nil {
		t.Fatalf("InspectRelationships failed: %v", err)
	}
	found := false
	for _, r := range rels {
		if r.Type != "" && r.Target != "" {
			found = true
		}
		if r.ID == "" {
			t.Errorf("relationship missing an Id: %+v", r)
		}
	}
	if !found {
		t.Fatalf("expected at least one relationship with a type and target, got: %+v", rels)
	}

	if _, err := InspectRelationships(filePath, "xl/worksheets/nope.xml"); err == nil {
		t.Fatal("expected an error for a part with no relationships file")
	}
}

func TestExtractDrawingShapes(t *testing.T) {
	filePath := newFixtureWithHardwareIcon(t)

	shapes, err := ExtractDrawingShapes(filePath, "Sheet1")
	if err != nil {
		t.Fatalf("ExtractDrawingShapes failed: %v", err)
	}
	primitives, _ := HardwareIconPrimitives(HardwareIconHinge)
	if len(shapes) != len(primitives) {
		t.Fatalf("expected %d shapes, got %d: %+v", len(primitives), len(shapes), shapes)
	}
	for i, shape := range shapes {
		if shape.Kind != "shape" {
			t.Errorf("shape %d: expected kind 'shape', got %q", i, shape.Kind)
		}
		if shape.PrstGeom != string(primitives[i].Shape) {
			t.Errorf("shape %d: expected prstGeom %q, got %q", i, primitives[i].Shape, shape.PrstGeom)
		}
		if shape.FromCell != "C5" {
			t.Errorf("shape %d: expected anchor C5, got %s", i, shape.FromCell)
		}
	}
}

func TestValidateTemplate(t *testing.T) {
	filePath := newFixtureWithHardwareIcon(t)

	results, err := ValidateTemplate(filePath, []string{"Sheet1", "NoSuchSheet"})
	if err != nil {
		t.Fatalf("ValidateTemplate failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].OK || results[0].Error != "" {
		t.Errorf("expected Sheet1 to validate OK, got: %+v", results[0])
	}
	if results[1].OK || results[1].Error == "" {
		t.Errorf("expected NoSuchSheet to fail validation, got: %+v", results[1])
	}
}
