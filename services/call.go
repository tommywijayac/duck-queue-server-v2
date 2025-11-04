package services

import (
	"context"
	"errors"

	"github.com/beego/beego/v2/core/logs"
	"github.com/tommywijayac/duck-queue-server-v2/backend/models"
)

// Let the CS call the number.. definitely not a pun
type CallService struct {
	roomService *RoomService
}

func NewCallService(roomService *RoomService) *CallService {
	cs := &CallService{
		roomService: roomService,
	}

	// start consumer
	if err := models.NewCallJobStream(); err != nil {
		logs.Critical("fail to create call job stream: %s", err.Error())
		panic(err)
	}
	go cs.read()

	return cs
}

func (cs *CallService) AddCallJob(ctx context.Context, job models.CallJob) error {
	room, err := cs.roomService.GetRoom(ctx, job.RoomID)
	if err != nil {
		return err
	}

	_, exists := room.Counters[job.CounterID]
	if !exists {
		return errors.New("counter not found in room")
	}

	return job.Add()
}

func (cs *CallService) doCallJob(ctx context.Context, job models.CallJob) error {
	// callback room service to handle queue
	// adjust queue first since it's fast, and to provide visual feedback first
	if err := cs.roomService.CallQueue(ctx, job.RoomID, job.CounterID, job.QueueNumber); err != nil {
		return err
	}

	// TOOD: integrate with notifier service
	return nil
}

func (cs *CallService) read() {
	for {
		ctx := context.Background()

		jobs, err := models.GetCallJobs(ctx)
		if err != nil {
			logs.Critical("fail to get call jobs: %s", err.Error())
			continue
		}

		for _, job := range jobs {
			if job.RoomID == "" ||
				job.CounterID == "" ||
				job.QueueNumber == "" {
				logs.Error("invalid call payload")
				continue
			}

			if err := cs.doCallJob(ctx, job); err != nil {
				logs.Error("fail to process call job: %s", err.Error())
				continue
			}

			if err := job.Done(ctx); err != nil {
				logs.Critical("fail to mark call job as finished: %s", err.Error())
				continue
			}
		}
	}
}
