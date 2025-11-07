package services

import (
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/tommywijayac/duck-queue-server-v2/models"
)

type PrinterService struct {
	LogoPath string
	Title    string
	Subtitle string
}

func NewPrinterService() *PrinterService {
	logo, err := web.AppConfig.String("printer::logo")
	if err != nil {
		logo = ""
	}
	title, err := web.AppConfig.String("printer::title")
	if err != nil {
		title = ""
	}
	subtitle, err := web.AppConfig.String("printer::subtitle")
	if err != nil {
		subtitle = ""
	}

	return &PrinterService{
		LogoPath: logo,
		Title:    title,
		Subtitle: subtitle,
	}
}

func (ps *PrinterService) PrintQueue(queueNumber string) error {
	builder := models.NewPrinterBuilder()

	if len(ps.LogoPath) > 0 {
		builder.AddImage(ps.LogoPath).AddSpace(2)
	}

	if len(ps.Title) > 0 {
		builder.AddText(ps.Title,
			models.WithPrinterLineStyle(models.FontStyle{Bold: true}),
		).AddSpace(1)
	}

	if len(ps.Subtitle) > 0 {
		builder.AddText(ps.Subtitle).AddSpace(1.5)
	}

	printer := builder.
		AddText(time.Now().Format("2 Jan 2006, 15:04:05"),
			models.WithPrinterLineSize(models.FontSize{Point: 8}),
		).
		AddSpace(1.5).
		AddText("your number is").
		AddSpace(1.5).
		AddText(queueNumber,
			models.WithPrinterLineSize(models.FontSize{Point: 36}),
			models.WithPrinterLineStyle(models.FontStyle{Bold: true, Underline: true})).
		AddSpace(2).
		AddText("please wait to be called").
		AddSpace(1).
		AddText("---").
		Build()

	return printer.Print()
}
