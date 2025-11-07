package controllers

import (
	"net/http"

	"github.com/beego/beego/v2/server/web"
	"github.com/tommywijayac/duck-queue-server-v2/models"
)

type RoomController struct {
	web.Controller
}

func (c *RoomController) CreateRoomQueue() {
	type Request struct {
		DestinationRoomID string `json:"destination_room_id"`
		Name              string `json:"name"`
		Phone             string `json:"phone"`
	}

	ctx := c.Ctx.Request.Context()
	roomID := c.Ctx.Input.Param(":id")

	var req Request
	if err := c.BindJSON(&req); err != nil {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{
			"error":       "Invalid input",
			"dev_message": err.Error(),
		}
		c.ServeJSON()
		return
	}

	createdQueue, err := RoomService.CreateQueue(ctx, roomID, req.DestinationRoomID, models.QueueInfo{
		Name:  req.Name,
		Phone: req.Phone,
	})
	if err != nil {
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{
			"error":       "Failed to create queue",
			"dev_message": err.Error(),
		}
		c.ServeJSON()
		return
	}

	c.Ctx.Output.SetStatus(http.StatusCreated)
	c.Data["json"] = map[string]interface{}{
		"message": "Queue created successfully",
		"queue":   createdQueue,
	}
	c.ServeJSON()
}

func (c *RoomController) ProcessRoomQueue() {
	type Request struct {
		OriginQueue string `json:"origin_queue"`
		CounterID   string `json:"counter_id"`
		QueueNumber string `json:"queue_number"`
	}

	ctx := c.Ctx.Request.Context()
	roomID := c.Ctx.Input.Param(":id")

	var req Request
	if err := c.BindJSON(&req); err != nil {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{
			"error":       "Invalid input",
			"dev_message": err.Error(),
		}
		c.ServeJSON()
		return
	}

	err := RoomService.ProcessQueue(ctx, roomID, req.OriginQueue, req.CounterID, req.QueueNumber)
	if err != nil {
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{
			"error":       "Failed to process queue",
			"dev_message": err.Error(),
		}
		c.ServeJSON()
		return
	}

	c.Ctx.Output.SetStatus(http.StatusOK)
	c.Data["json"] = map[string]interface{}{
		"message": "Queue processed successfully",
	}
	c.ServeJSON()
}

func (c *RoomController) SkipRoomQueue() {
	type Request struct {
		CounterID   string `json:"counter_id"`
		QueueNumber string `json:"queue_number"`
	}

	ctx := c.Ctx.Request.Context()
	roomID := c.Ctx.Input.Param(":id")

	var req Request
	if err := c.BindJSON(&req); err != nil {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{
			"error":       "Invalid input",
			"dev_message": err.Error(),
		}
		c.ServeJSON()
		return
	}

	err := RoomService.SkipQueue(ctx, roomID, req.CounterID, req.QueueNumber)
	if err != nil {
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{
			"error":       "Failed to skip queue",
			"dev_message": err.Error(),
		}
		c.ServeJSON()
		return
	}

	c.Ctx.Output.SetStatus(http.StatusOK)
	c.Data["json"] = map[string]interface{}{
		"message": "Queue skipped successfully",
	}
	c.ServeJSON()
}

func (c *RoomController) MoveRoomQueue() {
	type Request struct {
		DestinationRoomID string `json:"destination_room_id"`
		CounterID         string `json:"counter_id"`
		QueueNumber       string `json:"queue_number"`
	}

	ctx := c.Ctx.Request.Context()
	roomID := c.Ctx.Input.Param(":id")

	var req Request
	if err := c.BindJSON(&req); err != nil {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{
			"error":       "Invalid input",
			"dev_message": err.Error(),
		}
		c.ServeJSON()
		return
	}

	err := RoomService.MoveQueue(ctx, roomID, req.DestinationRoomID, req.CounterID, req.QueueNumber)
	if err != nil {
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{
			"error":       "Failed to move queue",
			"dev_message": err.Error(),
		}
		c.ServeJSON()
		return
	}

	c.Ctx.Output.SetStatus(http.StatusOK)
	c.Data["json"] = map[string]interface{}{
		"message": "Queue moved successfully",
	}
	c.ServeJSON()
}

func (c *RoomController) CallRoomQueue() {
	type Request struct {
		CounterID   string `json:"counter_id"`
		QueueNumber string `json:"queue_number"`
	}

	ctx := c.Ctx.Request.Context()
	roomID := c.Ctx.Input.Param(":id")

	var req Request
	if err := c.BindJSON(&req); err != nil {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{
			"error":       "Invalid input",
			"dev_message": err.Error(),
		}
		c.ServeJSON()
		return
	}

	if err := CallService.AddCallJob(ctx, models.CallJob{
		RoomID:      roomID,
		CounterID:   req.CounterID,
		QueueNumber: req.QueueNumber,
	}); err != nil {
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{
			"error":       "Failed to add call job",
			"dev_message": err.Error(),
		}
		c.ServeJSON()
		return
	}

	c.Ctx.Output.SetStatus(http.StatusAccepted)
	c.Data["json"] = map[string]interface{}{
		"message": "Queue job added successfully",
	}
	c.ServeJSON()
}

func (c *RoomController) GetRoomQueues() {
	ctx := c.Ctx.Request.Context()
	roomID := c.Ctx.Input.Param(":id")

	queues, err := RoomService.GetRoomQueues(ctx, roomID)
	if err != nil {
		c.Ctx.Output.SetStatus(http.StatusNotFound)
		c.Data["json"] = map[string]string{
			"error":       "Room not found",
			"dev_message": err.Error(),
		}
		c.ServeJSON()
		return
	}

	// return room information needed to construct view
	roomDetail, counters, err := RoomService.GetRoomDetails(ctx, roomID)
	if err != nil {
		c.Ctx.Output.SetStatus(http.StatusNotFound)
		c.Data["json"] = map[string]string{
			"error":       "Room not found",
			"dev_message": err.Error(),
		}
		c.ServeJSON()
		return
	}

	c.Ctx.Output.SetStatus(http.StatusOK)
	c.Data["json"] = map[string]interface{}{
		"details": map[string]interface{}{
			"room_id":   roomID,
			"room_name": roomDetail.Name,
			"counters":  counters,
		},
		"queues": queues,
	}
	c.ServeJSON()
}
