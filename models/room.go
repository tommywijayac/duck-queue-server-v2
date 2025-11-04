package models

import (
	"context"
	"errors"
	"fmt"
)

type Room struct {
	RoomDetail
	Id string

	// [main] -> [counter]
	// [skip] -> [counter]
	//
	// [counter] -> [call]
	// [counter] -> [skip]
	// [counter] -> [main] (other room)

	mainQueue    *Queue            // key is {room id}:main
	counterQueue map[string]*Queue // room counter id -> queue number. key is {room id}:counter:{counter id}
	skipQueue    *Queue            // key is {room id}:skip
}

type RoomDetail struct {
	Name     string                       `json:"name"`
	Actions  []RoomAllowedAction          `json:"actions"`
	Counters map[string]RoomCounterDetail `json:"counters"`
}

type RoomCounterDetail struct {
	DisplayName string `json:"name"`
}

type RoomAction string

const (
	RoomActionCreate RoomAction = "create"
	RoomActionMove   RoomAction = "move"
	RoomActionCall   RoomAction = "call"
)

type RoomAllowedAction struct {
	Action             RoomAction `json:"action"`
	DestinationRoomIDs []string   `json:"destination_ids"`
}

const (
	InternalRoomIDDisplay = "DISPLAY"
)

func NewRoom(id string, detail RoomDetail) *Room {
	counterQueue := make(map[string]*Queue)
	for cid := range detail.Counters {
		counterQueueCfg := DefaultQueueCfg
		counterQueueCfg.MaxQueue = 1
		counterQueue[cid] = NewQueue(fmt.Sprintf("%s:counter:%s", id, cid), counterQueueCfg)
	}

	return &Room{
		RoomDetail: detail,
		Id:         id,

		mainQueue:    NewQueue(id+":main", DefaultQueueCfg),
		counterQueue: counterQueue,
		skipQueue:    NewQueue(id+":skip", DefaultQueueCfg),
	}
}

// CreateQueue creates a new queue. By default, it's appended to the main queue.
func (r *Room) CreateQueue(ctx context.Context, info QueueInfo) (QueueItem, error) {
	return r.mainQueue.Create(ctx, r.Id, info)
}

// ProcessQueue moves a queue from main OR skip queue to counter queue.
func (r *Room) ProcessQueue(ctx context.Context, originQueue string, counterId, queueNumber string) error {
	switch originQueue {
	case "main":
		return r.mainQueue.Move(ctx, queueNumber, r.counterQueue[counterId])
	case "skip":
		return r.skipQueue.Move(ctx, queueNumber, r.counterQueue[counterId])
	}
	return errors.New("invalid origin queue")
}

// SkipQueue moves a queue from counter queue to skip queue.
func (r *Room) SkipQueue(ctx context.Context, counterId, queueNumber string) error {
	return r.counterQueue[counterId].Move(ctx, queueNumber, r.skipQueue)
}

// MoveQueue moves a queue from counter queue to another room's main queue. It doesn't create a new queue number
func (r *Room) MoveQueue(ctx context.Context, counterId, queueNumber string, destination *Room) error {
	return r.counterQueue[counterId].Move(ctx, queueNumber, destination.mainQueue)
}

func (r *Room) GetQueues(ctx context.Context) (map[string][]QueueItem, error) {
	queues := make(map[string][]QueueItem)

	mainItems, err := r.mainQueue.List(ctx)
	if err != nil {
		return nil, err
	}
	queues["main"] = mainItems

	skipItems, err := r.skipQueue.List(ctx)
	if err != nil {
		return nil, err
	}
	queues["skip"] = skipItems

	for counterId, counterQueue := range r.counterQueue {
		counterItems, err := counterQueue.List(ctx)
		if err != nil {
			return nil, err
		}
		queues[counterId] = counterItems
	}

	return queues, nil
}
