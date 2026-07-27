package excel

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/xuri/excelize/v2"
)

// HardwareIconVerification is the outcome of independently re-reading the
// OOXML drawing part of a saved file to confirm a HardwareIconResult was
// actually persisted, rather than trusting that DrawHardwareIcon/Save
// succeeding without error means the file is correct.
type HardwareIconVerification struct {
	Verified           bool
	AnchorCell         string
	ExpectedShapeTypes []string
	// FoundShapeTypes are the prstGeom shape types excelize/Excel actually
	// wrote at the icon's anchor cell, in document order.
	FoundShapeTypes []string
	Message         string
}

// VerifyHardwareIcon re-opens the saved .xlsx file as a raw zip archive and
// parses its real DrawingML XML to confirm the icon described by result was
// written to disk: the right number of shapes, of the right preset
// geometries, anchored at the right cell. It works for files saved by either
// backend (excelize or OLE/real Excel) because both ultimately persist the
// same OOXML drawing part format - this function never trusts in-memory
// state, only the bytes that were actually saved.
func VerifyHardwareIcon(fileAbsolutePath string, result *HardwareIconResult) (*HardwareIconVerification, error) {
	zr, err := zip.OpenReader(fileAbsolutePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open saved file for verification: %w", err)
	}
	defer zr.Close()

	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		files[f.Name] = f
	}

	worksheetPath, err := findWorksheetPath(files, result.SheetName)
	if err != nil {
		return nil, err
	}

	drawingPath, err := findDrawingPath(files, worksheetPath)
	if err != nil {
		return &HardwareIconVerification{
			Verified:           false,
			AnchorCell:         result.AnchorCell,
			ExpectedShapeTypes: result.ExpectedShapeTypes,
			Message:            fmt.Sprintf("no drawing found for sheet %q: %s", result.SheetName, err.Error()),
		}, nil
	}

	drawingData, err := readZipFile(files, drawingPath)
	if err != nil {
		return nil, err
	}

	var drawing drawingWsDrXML
	if err := xml.Unmarshal(drawingData, &drawing); err != nil {
		return nil, fmt.Errorf("failed to parse drawing XML %q: %w", drawingPath, err)
	}

	anchorCol, anchorRow, err := excelize.CellNameToCoordinates(result.AnchorCell)
	if err != nil {
		return nil, fmt.Errorf("invalid anchor cell %q: %w", result.AnchorCell, err)
	}
	// OOXML from/col and from/row are 0-indexed; CellNameToCoordinates is 1-indexed.
	anchorCol--
	anchorRow--

	var matchedTypes []string
	for _, anchor := range drawing.TwoCellAnchors {
		if anchor.From.Col == anchorCol && anchor.From.Row == anchorRow && anchor.Sp != nil {
			matchedTypes = append(matchedTypes, anchor.Sp.SpPr.PrstGeom.Prst)
		}
	}

	verification := &HardwareIconVerification{
		AnchorCell:         result.AnchorCell,
		ExpectedShapeTypes: result.ExpectedShapeTypes,
		FoundShapeTypes:    matchedTypes,
	}

	// AddShape appends to the end of the drawing document, so the most
	// recently drawn icon's shapes are the tail of the shapes found at this
	// anchor cell (earlier icons redrawn at the same cell, if any, precede
	// them).
	if len(matchedTypes) >= len(result.ExpectedShapeTypes) {
		tail := matchedTypes[len(matchedTypes)-len(result.ExpectedShapeTypes):]
		if equalStringSlices(tail, result.ExpectedShapeTypes) {
			verification.Verified = true
			verification.Message = fmt.Sprintf("verified %d shape(s) for icon at %s", len(result.ExpectedShapeTypes), result.AnchorCell)
			return verification, nil
		}
	}

	verification.Message = fmt.Sprintf(
		"expected shapes %v at cell %s but found %v",
		result.ExpectedShapeTypes, result.AnchorCell, matchedTypes,
	)
	return verification, nil
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func readZipFile(files map[string]*zip.File, name string) ([]byte, error) {
	f, ok := files[name]
	if !ok {
		return nil, fmt.Errorf("part not found in file: %s", name)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open part %s: %w", name, err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read part %s: %w", name, err)
	}
	return data, nil
}

// findWorksheetPath resolves a sheet name (e.g. "Sheet1") to its worksheet
// part path (e.g. "xl/worksheets/sheet1.xml") by reading the real
// xl/workbook.xml and xl/_rels/workbook.xml.rels parts from the saved file.
func findWorksheetPath(files map[string]*zip.File, sheetName string) (string, error) {
	workbookData, err := readZipFile(files, "xl/workbook.xml")
	if err != nil {
		return "", err
	}
	var workbook workbookXML
	if err := xml.Unmarshal(workbookData, &workbook); err != nil {
		return "", fmt.Errorf("failed to parse xl/workbook.xml: %w", err)
	}

	var rID string
	for _, sheet := range workbook.Sheets {
		if sheet.Name == sheetName {
			rID = sheet.RID
			break
		}
	}
	if rID == "" {
		return "", fmt.Errorf("sheet not found in workbook: %s", sheetName)
	}

	relsData, err := readZipFile(files, "xl/_rels/workbook.xml.rels")
	if err != nil {
		return "", err
	}
	var rels relationshipsXML
	if err := xml.Unmarshal(relsData, &rels); err != nil {
		return "", fmt.Errorf("failed to parse xl/_rels/workbook.xml.rels: %w", err)
	}
	for _, rel := range rels.Relationships {
		if rel.ID == rID {
			return path.Join("xl", rel.Target), nil
		}
	}
	return "", fmt.Errorf("relationship not found for sheet: %s", sheetName)
}

// findDrawingPath resolves a worksheet part path (e.g.
// "xl/worksheets/sheet1.xml") to its drawing part path (e.g.
// "xl/drawings/drawing1.xml") by reading the worksheet's own real .rels
// part from the saved file.
func findDrawingPath(files map[string]*zip.File, worksheetPath string) (string, error) {
	dir, base := path.Split(worksheetPath)
	relsPath := path.Join(dir, "_rels", base+".rels")

	relsData, err := readZipFile(files, relsPath)
	if err != nil {
		return "", err
	}
	var rels relationshipsXML
	if err := xml.Unmarshal(relsData, &rels); err != nil {
		return "", fmt.Errorf("failed to parse %s: %w", relsPath, err)
	}
	for _, rel := range rels.Relationships {
		if strings.HasSuffix(rel.Type, "/drawing") {
			return path.Join(dir, rel.Target), nil
		}
	}
	return "", fmt.Errorf("no drawing relationship found in %s", relsPath)
}

type workbookXML struct {
	Sheets []workbookSheetXML `xml:"sheets>sheet"`
}

type workbookSheetXML struct {
	Name string `xml:"name,attr"`
	RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
}

type relationshipsXML struct {
	Relationships []relationshipXML `xml:"Relationship"`
}

type relationshipXML struct {
	ID     string `xml:"Id,attr"`
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
}

type drawingWsDrXML struct {
	TwoCellAnchors []drawingAnchorXML `xml:"twoCellAnchor"`
}

type drawingAnchorXML struct {
	From drawingFromXML `xml:"from"`
	Sp   *drawingSpXML  `xml:"sp"`
}

type drawingFromXML struct {
	Col int `xml:"col"`
	Row int `xml:"row"`
}

type drawingSpXML struct {
	SpPr drawingSpPrXML `xml:"spPr"`
}

type drawingSpPrXML struct {
	PrstGeom drawingPrstGeomXML `xml:"prstGeom"`
}

type drawingPrstGeomXML struct {
	Prst string `xml:"prst,attr"`
}
