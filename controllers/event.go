package controllers

func (c *RoomController) StreamRoomEvents() {
	roomID := c.Ctx.Input.Param(":id")
	clientID := c.Ctx.Input.Query(":client_id")

	c.Ctx.Output.Header("Content-Type", "text/event-stream")
	c.Ctx.Output.Header("Cache-Control", "no-cache")
	c.Ctx.Output.Header("Connection", "keep-alive")
	c.Ctx.Output.Header("Access-Control-Allow-Origin", "*")
	c.Ctx.Output.Header("Access-Control-Allow-Headers", "Cache-Control")

	EventHubService.RegisterClient(roomID, clientID, c.Ctx.ResponseWriter)
	<-c.Ctx.Request.Context().Done()
	EventHubService.UnregisterClient(roomID, clientID)
}
