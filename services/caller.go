package services

import (
	"context"
	"errors"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/tommywijayac/duck-queue-server-v2/models"
)

// Let the CS call the number.. definitely not a pun
type CallService struct {
	// right now call job streams only has one worker group, which means limited to one physical speaker
	// if want to support multiple speakers, need to create multiple job streams
	callQueue *models.CallQueue

	// dependencies
	roomService *RoomService
}

func NewCallService(roomService *RoomService) *CallService {
	callQueue, err := models.NewCallQueue("default")
	if err != nil {
		logs.Critical("fail to create call queue: %s", err.Error())
		panic(err)
	}

	cs := &CallService{
		callQueue:   callQueue,
		roomService: roomService,
	}

	// start consumer
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

	// hydrate details so worker can simply use info inside the job
	job.RoomName = room.Name
	job.CounterName = room.Counters[job.CounterID].DisplayName

	return cs.callQueue.Addjob(ctx, &job)
}

func (cs *CallService) doCallJob(ctx context.Context, job *models.CallJob) error {
	// hydrate more details for log
	// log first so UI can display immediately
	job.CalledAt = time.Now()
	if err := cs.callQueue.Log(ctx, job); err != nil {
		return err
	}

	// TODO: send visual cue to client UI
	// TODO: send audio cue to speaker device
	logs.Debug("call job received with details: ", job)

	return nil
}

func (cs *CallService) read() {
	for {
		ctx := context.Background()

		jobs, err := cs.callQueue.GetCallJobs(ctx)
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

			if err := cs.doCallJob(ctx, &job); err != nil {
				logs.Error("fail to process call job: %s", err.Error())
				continue
			}

			if err := cs.callQueue.Done(ctx, &job); err != nil {
				logs.Critical("fail to mark call job as finished: %s", err.Error())
				continue
			}
		}
	}
}
