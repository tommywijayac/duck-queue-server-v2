package controllers

import (
	"time"

	"github.com/tommywijayac/duck-queue-server-v2/backend/services"
)

var (
	loc             *time.Location
	RoomService     *services.RoomService
	CallService     *services.CallService
	PrinterService  *services.PrinterService
	EventHubService *services.EventHubService
)

func Init() {
	var err error
	loc, err = time.LoadLocation("Asia/Jakarta")
	if err != nil {
		panic(err)
	}

	PrinterService = services.NewPrinterService()
	RoomService = services.NewRoomService(PrinterService)
	CallService = services.NewCallService(RoomService)
	EventHubService = services.NewEventHubService()
}
