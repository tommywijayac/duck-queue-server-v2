package routers

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/tommywijayac/duck-queue-server-v2/backend/controllers"
)

func Init() {
	// Mutation
	web.Router("/api/rooms/:id", &controllers.RoomController{}, "post:CreateRoomQueue")
	web.Router("/api/rooms/:id/process", &controllers.RoomController{}, "post:ProcessRoomQueue")
	web.Router("/api/rooms/:id/skip", &controllers.RoomController{}, "post:SkipRoomQueue")
	web.Router("/api/rooms/:id/move", &controllers.RoomController{}, "post:MoveRoomQueue")
	web.Router("/api/rooms/:id/call", &controllers.RoomController{}, "post:CallRoomQueue")

	web.Router("/api/dispenser/exit", &controllers.AdminController{}, "post:ExitDispenserApp")

	// Query
	web.Router("/api/rooms/:id", &controllers.RoomController{}, "get:GetRoomQueues")
	// web.Router("/api/rooms", &controllers.RoomController{}, "get:ListRooms")
}
