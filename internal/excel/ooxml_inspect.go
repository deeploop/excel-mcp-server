package excel

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"path"

	"github.com/xuri/excelize/v2"
)

// OoxmlRelationship is one <Relationship> entry from a .rels part.
type OoxmlRelationship struct {
	ID     string
	Type   string
	Target string
}

func openZipIndex(fileAbsolutePath string) (*zip.ReadCloser, map[string]*zip.File, error) {
	zr, err := zip.OpenReader(fileAbsolutePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		files[f.Name] = f
	}
	return zr, files, nil
}

// InspectRelationships lists every relationship defined by the .rels part
// belonging to the given OOXML part path (e.g. "xl/worksheets/sheet1.xml"),
// so a broken template's relationship chain can be diagnosed directly
// instead of guessed at by hand.
func InspectRelationships(fileAbsolutePath string, partPath string) ([]OoxmlRelationship, error) {
	zr, files, err := openZipIndex(fileAbsolutePath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	dir, base := path.Split(partPath)
	relsPath := path.Join(dir, "_rels", base+".rels")

	data, err := readZipFile(files, relsPath)
	if err != nil {
		return nil, fmt.Errorf("no relationships found for part %q (looked for %q): %w", partPath, relsPath, err)
	}
	var rels relationshipsXML
	if err := xml.Unmarshal(data, &rels); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", relsPath, err)
	}

	result := make([]OoxmlRelationship, len(rels.Relationships))
	for i, r := range rels.Relationships {
		result[i] = OoxmlRelationship{ID: r.ID, Type: r.Type, Target: r.Target}
	}
	return result, nil
}

// ResolvedSheetPart is the location of a worksheet's own OOXML part and its
// relationships part within the workbook's zip archive.
type ResolvedSheetPart struct {
	WorksheetPath string
	RelsPath      string
}

// ResolveSheetPart resolves a sheet name to its worksheet part path (e.g.
// "xl/worksheets/sheet1.xml") and relationships part path, by walking the
// xl/workbook.xml -> xl/_rels/workbook.xml.rels chain - the same chain
// excel_draw_hardware_icon's automatic verification walks.
func ResolveSheetPart(fileAbsolutePath string, sheetName string) (*ResolvedSheetPart, error) {
	zr, files, err := openZipIndex(fileAbsolutePath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	worksheetPath, err := findWorksheetPath(files, sheetName)
	if err != nil {
		return nil, err
	}
	dir, base := path.Split(worksheetPath)
	relsPath := path.Join(dir, "_rels", base+".rels")

	return &ResolvedSheetPart{WorksheetPath: worksheetPath, RelsPath: relsPath}, nil
}

// DrawingShape is one shape or picture anchored on a sheet's DrawingML part.
type DrawingShape struct {
	// Kind is "shape" (a vector prstGeom shape) or "picture".
	Kind string
	// PrstGeom is the preset geometry name (e.g. "rect"), empty for pictures.
	PrstGeom string
	// FromCell/ToCell are the anchor's cell-range corners in A1 notation
	// (e.g. "B2" / "E10"), i.e. the same information excel_draw_hardware_icon
	// checks when verifying an icon's anchor.
	FromCell string
	ToCell   string
}

type drawingAnchorFullXML struct {
	From drawingFromXML `xml:"from"`
	To   drawingFromXML `xml:"to"`
	Sp   *drawingSpXML  `xml:"sp"`
	Pic  *struct{}      `xml:"pic"`
}

type drawingWsDrFullXML struct {
	TwoCellAnchors []drawingAnchorFullXML `xml:"twoCellAnchor"`
}

// ExtractDrawingShapes walks the sheet.xml -> sheet.xml.rels -> drawingN.xml
// relationship chain and returns every shape/picture anchored on the sheet,
// with their cell-range anchors already resolved to A1 notation (this
// folds in what would otherwise be a separate "get drawing anchors" call).
func ExtractDrawingShapes(fileAbsolutePath string, sheetName string) ([]DrawingShape, error) {
	zr, files, err := openZipIndex(fileAbsolutePath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	worksheetPath, err := findWorksheetPath(files, sheetName)
	if err != nil {
		return nil, err
	}
	drawingPath, err := findDrawingPath(files, worksheetPath)
	if err != nil {
		return nil, fmt.Errorf("no drawing found for sheet %q: %w", sheetName, err)
	}
	data, err := readZipFile(files, drawingPath)
	if err != nil {
		return nil, err
	}
	var drawing drawingWsDrFullXML
	if err := xml.Unmarshal(data, &drawing); err != nil {
		return nil, fmt.Errorf("failed to parse drawing XML %q: %w", drawingPath, err)
	}

	shapes := make([]DrawingShape, 0, len(drawing.TwoCellAnchors))
	for _, anchor := range drawing.TwoCellAnchors {
		if anchor.Sp == nil && anchor.Pic == nil {
			continue
		}
		fromCell, err := excelize.CoordinatesToCellName(anchor.From.Col+1, anchor.From.Row+1)
		if err != nil {
			return nil, err
		}
		toCell, err := excelize.CoordinatesToCellName(anchor.To.Col+1, anchor.To.Row+1)
		if err != nil {
			return nil, err
		}
		shape := DrawingShape{FromCell: fromCell, ToCell: toCell}
		if anchor.Sp != nil {
			shape.Kind = "shape"
			shape.PrstGeom = anchor.Sp.SpPr.PrstGeom.Prst
		} else {
			shape.Kind = "picture"
		}
		shapes = append(shapes, shape)
	}
	return shapes, nil
}

// TemplateSheetValidation is the validation outcome for one required sheet.
type TemplateSheetValidation struct {
	SheetName string
	OK        bool
	Error     string
}

// ValidateTemplate confirms that every sheet in requiredSheets exists in the
// workbook and that its relationship chain resolves to a real worksheet
// part, so a template that has suffered tab-index or relationship
// corruption is caught before it's used, rather than failing confusingly
// partway through a later drawing/write operation.
func ValidateTemplate(fileAbsolutePath string, requiredSheets []string) ([]TemplateSheetValidation, error) {
	zr, files, err := openZipIndex(fileAbsolutePath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	results := make([]TemplateSheetValidation, 0, len(requiredSheets))
	for _, sheetName := range requiredSheets {
		worksheetPath, err := findWorksheetPath(files, sheetName)
		if err != nil {
			results = append(results, TemplateSheetValidation{SheetName: sheetName, OK: false, Error: err.Error()})
			continue
		}
		if _, ok := files[worksheetPath]; !ok {
			results = append(results, TemplateSheetValidation{
				SheetName: sheetName, OK: false,
				Error: fmt.Sprintf("worksheet part missing from archive: %s", worksheetPath),
			})
			continue
		}
		results = append(results, TemplateSheetValidation{SheetName: sheetName, OK: true})
	}
	return results, nil
}
