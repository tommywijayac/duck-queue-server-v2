package models

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tommywijayac/duck-queue-server-v2/backend/databases"
)

const (
	streamCallJob       string = "call_job"
	streamCallJobWorker string = "call_job_cg"
)

func NewCallJobStream() error {
	err := databases.RedisClient.XGroupCreateMkStream(context.Background(), streamCallJob, streamCallJobWorker, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

type CallJob struct {
	// assigned automatically by redis stream
	ID string

	// call destination
	RoomName    string // needed?
	RoomID      string
	CounterID   string
	CounterName string // needed?
	QueueNumber string
}

func (job *CallJob) Add() error {
	var ctx context.Context = context.Background()

	// perform trimming to maintain size
	// default stream-node-max-entries is 100, so once entries exceed this limit
	// subsequent add will trim to max len.
	// in other words, need to have ~90 pending call jobs before trimming corrupts the job queue.
	// which should be impossible in this case.
	return databases.RedisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: streamCallJob,
		ID:     "",
		MaxLen: 10,
		Approx: true,
		Values: map[string]interface{}{
			"room_name":    job.RoomName,
			"room_id":      job.RoomID,
			"counter_name": job.CounterName,
			"counter_id":   job.CounterID,
			"queue_number": job.QueueNumber,
		},
	}).Err()
}

func GetCallJobs(ctx context.Context) ([]CallJob, error) {
	entries, err := databases.RedisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    streamCallJobWorker,
		Consumer: fmt.Sprintf("%d", time.Now().UnixMilli()),
		Streams:  []string{streamCallJob, ">"},
		Count:    1,
		Block:    0,
		NoAck:    false,
	}).Result()
	if err != nil {
		return nil, err
	}

	var jobs []CallJob
	for i := 0; i < len(entries); i++ {
		values := entries[0].Messages[i].Values
		jobs = append(jobs, CallJob{
			ID:          entries[0].Messages[i].ID,
			RoomName:    getValues(values, "room_name"),
			RoomID:      getValues(values, "room_id"),
			CounterName: getValues(values, "counter_name"),
			CounterID:   getValues(values, "counter_id"),
			QueueNumber: getValues(values, "queue_number"),
		})
	}

	return jobs, nil
}

func getValues(values map[string]interface{}, key string) string {
	v, ok := values[key]
	if !ok {
		return ""
	}
	vv, ok := v.(string)
	if !ok {
		return ""
	}
	return vv
}

func (job *CallJob) Done(ctx context.Context) error {
	return databases.RedisClient.XAck(ctx, streamCallJob, streamCallJobWorker, job.ID).Err()
}

// histories
