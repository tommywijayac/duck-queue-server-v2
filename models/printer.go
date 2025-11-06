package models

import (
	"errors"
	"fmt"
	"math"
	"os/exec"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/jung-kurt/gofpdf"
)

const (
	// printMethodLP prints by creating temp .pdf file and print using lp command.
	// However, this method only works if the printer is located at server.
	printMethodLP = "lp"
	// PrintMethodPOS  = "pos"
	// PrintMethodIPP  = "ipp"

	contentText  = "text"
	contentImage = "image"
)

type PrinterBuilder struct {
	lines []printerLine
}

type Printer struct {
	isEnabled bool
	method    string
	lines     []printerLine
}

type printerLine struct {
	Content string

	Text  string
	Size  FontSize
	Style FontStyle

	ImagePath string

	// Spacing in N multiplier. 1N = 8 Points = 8 * 0.35278 mm
	SpacingN float64
}

type LineOptions func(*printerLine)

func WithPrinterLineSize(size FontSize) LineOptions {
	return func(pl *printerLine) {
		pl.Size = size
	}
}

func WithPrinterLineStyle(style FontStyle) LineOptions {
	return func(pl *printerLine) {
		pl.Style = style
	}
}

type FontSize struct {
	// 1 Point = 0.35278 mm
	Point int
	// POSWidth  uint8
	// POSHeight uint8
}

type FontStyle struct {
	Bold      bool
	Underline bool
}

func NewPrinterBuilder() *PrinterBuilder {
	return &PrinterBuilder{
		lines: make([]printerLine, 0),
	}
}

func (pb *PrinterBuilder) addLine(line printerLine) *PrinterBuilder {
	pb.lines = append(pb.lines, line)
	return pb
}

func (pb *PrinterBuilder) AddText(text string, opts ...LineOptions) *PrinterBuilder {
	pl := printerLine{
		Content:  contentText,
		Text:     text,
		Size:     FontSize{Point: 10}, // default
		SpacingN: 1,                   // default
	}
	for _, opt := range opts {
		opt(&pl)
	}
	return pb.addLine(pl)
}

func (pb *PrinterBuilder) AddImage(imagePath string) *PrinterBuilder {
	return pb.addLine(printerLine{
		Content:   contentImage,
		ImagePath: imagePath,
		SpacingN:  1,
	})
}

func (pb *PrinterBuilder) AddSpace(spacingN float64) *PrinterBuilder {
	return pb.addLine(printerLine{
		Content:  contentText,
		Text:     "",
		SpacingN: spacingN,
	})
}

func (pb *PrinterBuilder) Build() *Printer {
	isEnabled, err := web.AppConfig.Bool("printer::enable")
	if err != nil {
		isEnabled = false
	}

	method, err := web.AppConfig.String("printer::method")
	if err != nil {
		isEnabled = false
	}

	return &Printer{
		isEnabled: isEnabled,
		method:    method,
		lines:     pb.lines,
	}
}

func (p *Printer) Print() error {
	if !p.isEnabled {
		logs.Info("Printer disabled, skipping print")
		return nil
	}

	switch p.method {
	case printMethodLP:
		return p.printLP()
	default:
		return errors.New("printer: unsupported print method")
	}
}

// printLP prints lines to pdf file and send to printer using lp command
func (p *Printer) printLP() error {
	// subsequent file will overwrite previous file
	outPath := "tmp_queue.pdf"
	if err := p.printPDF(outPath); err != nil {
		return err
	}

	cmd := exec.Command("lp", outPath)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("fail to send lp command: %w", err)
	}

	return nil
}

// other methods such as POS printer, IPP printer can be added here

const (
	//ref: https://pkg.go.dev/github.com/jung-kurt/gofpdf@v1.16.2#Fpdf.CellFormat
	pageWidth  = 80.0 //mm
	pageHeight = 70.0 //mm
	alignment  = "CB" //h: center. y: baseline
	border     = ""
	line       = 1 //beginning of next line
	fill       = false

	defaultFont     = "helvetica"
	defaultFontSize = 12 //in points
	defaultSpacing  = 8  //in points
)

func (p *Printer) printPDF(outPath string) error {
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		OrientationStr: gofpdf.OrientationPortrait,
		UnitStr:        gofpdf.UnitMillimeter,
		Size: gofpdf.SizeType{
			Wd: pageWidth,
			Ht: pageHeight,
		},
	})

	pdf.AddPage()

	// tackle weird behavior it automatically set (x,y) to (10.00125, 10.00125)
	pdf.SetXY(0, 0)

	// mandatory as fallback
	pdf.SetFont(defaultFont, "", defaultFontSize)

	// mandatory to make center alignment work properly
	pdf.SetMargins(0, 0, 0)
	pdf.SetCellMargin(0)

	// idk why
	pdf.SetAutoPageBreak(false, 0)

	for _, l := range p.lines {
		switch l.Content {
		case contentText:
			// set font according to line
			style := ""
			if l.Style.Bold {
				style += "B"
			}
			if l.Style.Underline {
				style += "U"
			}
			pdf.SetFont(defaultFont, style, float64(l.Size.Point))

			pdf.CellFormat(
				pageWidth,
				pointToMilimeter(float64(l.Size.Point)),
				l.Text,
				border,
				line,
				alignment,
				fill,
				0, "",
			)

		case contentImage:
			pdf.ImageOptions(
				l.ImagePath,
				20, 0,
				40, 0,
				true,
				gofpdf.ImageOptions{},
				0, "",
			)
		}

		// if call pdf.Ln(0), then next line directly printed below the previous line
		// already with sufficient spacing
		sn := l.SpacingN - 1
		if sn < 0 {
			sn = 1
		}
		pdf.Ln(pointToMilimeter(float64(defaultSpacing * sn)))
	}

	return pdf.OutputFileAndClose(outPath)
}

func pointToMilimeter(point float64) float64 {
	return math.Round(point * 0.35278)
}
