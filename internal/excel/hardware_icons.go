package excel

import "fmt"

// HardwareIconType represents a predefined door/window hardware icon that can
// be drawn onto a worksheet as a small group of vector shapes.
type HardwareIconType string

const (
	HardwareIconHinge        HardwareIconType = "hinge"
	HardwareIconDoorHandle   HardwareIconType = "doorHandle"
	HardwareIconDeadbolt     HardwareIconType = "deadbolt"
	HardwareIconLockCylinder HardwareIconType = "lockCylinder"
	HardwareIconDoorCloser   HardwareIconType = "doorCloser"
	HardwareIconPeephole     HardwareIconType = "peephole"
)

func (t HardwareIconType) String() string {
	return string(t)
}

func (t HardwareIconType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func HardwareIconTypeValues() []HardwareIconType {
	return []HardwareIconType{
		HardwareIconHinge,
		HardwareIconDoorHandle,
		HardwareIconDeadbolt,
		HardwareIconLockCylinder,
		HardwareIconDoorCloser,
		HardwareIconPeephole,
	}
}

// PrimitiveShapeType is one of a small subset of OOXML DrawingML preset
// geometries ("prstGeom") used to compose a hardware icon.
//
// This subset was chosen deliberately narrow because it is the intersection
// of what excelize's AddShape can write (cross-platform backend) and what
// real Excel writes to the very same OOXML "prstGeom" attribute when a shape
// is created via Shapes.AddShape through OLE automation (Windows backend,
// see primitiveShapeToMsoAutoShapeType in excel_ole.go). Both backends
// therefore produce byte-identical shape type strings in the saved file,
// which lets VerifyHardwareIcon check either backend's output the same way.
type PrimitiveShapeType string

const (
	PrimitiveRect      PrimitiveShapeType = "rect"
	PrimitiveRoundRect PrimitiveShapeType = "roundRect"
	PrimitiveEllipse   PrimitiveShapeType = "ellipse"
)

// IconPrimitive is one vector shape making up part of a hardware icon.
// Bounds are fractions (0..1) of the icon's bounding square, relative to its
// top-left corner, so the whole icon can be scaled to any requested size.
type IconPrimitive struct {
	Shape     PrimitiveShapeType
	X0, Y0    float64
	X1, Y1    float64
	FillColor string // hex color, without a leading '#'
	LineColor string // hex color, without a leading '#'
}

// hardwareIconLibrary defines the primitives for each supported icon type.
// Order matters: it is the order shapes are drawn in, and the order
// VerifyHardwareIcon expects to find them back in the saved file.
var hardwareIconLibrary = map[HardwareIconType][]IconPrimitive{
	HardwareIconHinge: {
		{Shape: PrimitiveRoundRect, X0: 0.05, Y0: 0.05, X1: 0.45, Y1: 0.95, FillColor: "B0B0B0", LineColor: "555555"},
		{Shape: PrimitiveRoundRect, X0: 0.55, Y0: 0.05, X1: 0.95, Y1: 0.95, FillColor: "B0B0B0", LineColor: "555555"},
		{Shape: PrimitiveEllipse, X0: 0.42, Y0: 0.40, X1: 0.58, Y1: 0.60, FillColor: "555555", LineColor: "333333"},
	},
	HardwareIconDoorHandle: {
		{Shape: PrimitiveRoundRect, X0: 0.35, Y0: 0.05, X1: 0.65, Y1: 0.95, FillColor: "C0C0C0", LineColor: "555555"},
		{Shape: PrimitiveRoundRect, X0: 0.05, Y0: 0.42, X1: 0.65, Y1: 0.58, FillColor: "C0C0C0", LineColor: "555555"},
	},
	HardwareIconDeadbolt: {
		{Shape: PrimitiveRect, X0: 0.25, Y0: 0.05, X1: 0.75, Y1: 0.95, FillColor: "8C8C8C", LineColor: "444444"},
		{Shape: PrimitiveEllipse, X0: 0.35, Y0: 0.28, X1: 0.65, Y1: 0.58, FillColor: "D4AF37", LineColor: "8A6D14"},
		{Shape: PrimitiveRect, X0: 0.40, Y0: 0.65, X1: 0.60, Y1: 0.90, FillColor: "444444", LineColor: "222222"},
	},
	HardwareIconLockCylinder: {
		{Shape: PrimitiveEllipse, X0: 0.10, Y0: 0.10, X1: 0.90, Y1: 0.90, FillColor: "C0C0C0", LineColor: "555555"},
		{Shape: PrimitiveRect, X0: 0.45, Y0: 0.30, X1: 0.55, Y1: 0.70, FillColor: "222222", LineColor: "000000"},
	},
	HardwareIconDoorCloser: {
		{Shape: PrimitiveRect, X0: 0.05, Y0: 0.35, X1: 0.55, Y1: 0.65, FillColor: "6E6E6E", LineColor: "333333"},
		{Shape: PrimitiveRect, X0: 0.50, Y0: 0.45, X1: 0.95, Y1: 0.55, FillColor: "9E9E9E", LineColor: "333333"},
	},
	HardwareIconPeephole: {
		{Shape: PrimitiveEllipse, X0: 0.15, Y0: 0.15, X1: 0.85, Y1: 0.85, FillColor: "C0C0C0", LineColor: "555555"},
		{Shape: PrimitiveEllipse, X0: 0.35, Y0: 0.35, X1: 0.65, Y1: 0.65, FillColor: "111111", LineColor: "000000"},
	},
}

// HardwareIconPrimitives returns the ordered list of primitives that compose
// the given icon type, or an error if the icon type is unknown.
func HardwareIconPrimitives(iconType HardwareIconType) ([]IconPrimitive, error) {
	primitives, ok := hardwareIconLibrary[iconType]
	if !ok {
		return nil, fmt.Errorf("unknown hardware icon type: %s", iconType)
	}
	return primitives, nil
}

// HardwareIconOptions customizes how an icon is drawn.
type HardwareIconOptions struct {
	// SizePoints is the width/height of the icon's bounding square, in
	// points. Defaults to 36pt (0.5 inch) when zero.
	SizePoints float64
}

// HardwareIconResult describes what was actually drawn, so it can be
// independently re-verified against the saved file by VerifyHardwareIcon.
type HardwareIconResult struct {
	SheetName          string
	AnchorCell         string
	IconType           HardwareIconType
	ExpectedShapeTypes []string
}
