package models

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/redis/go-redis/v9"
	"github.com/tommywijayac/duck-queue-server-v2/backend/databases"
)

const (
	streamCallJob       string = "call_job"
	streamCallJobWorker string = "call_job_cg"
)

type CallQueue struct {
	id     string
	stream string
	worker string
}

type CallJob struct {
	// assigned automatically by redis stream
	ID string

	// call destination
	RoomName    string
	RoomID      string
	CounterID   string
	CounterName string
	QueueNumber string

	CalledAt time.Time
}

func NewCallQueue(id string) (*CallQueue, error) {
	stream := fmt.Sprintf("%s_%s", streamCallJob, id)
	worker := fmt.Sprintf("%s_%s", streamCallJobWorker, id)

	err := databases.RedisClient.XGroupCreateMkStream(context.Background(), stream, worker, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return nil, err
	}

	return &CallQueue{
		id:     id,
		stream: stream,
		worker: worker,
	}, nil
}

func (cq *CallQueue) Addjob(ctx context.Context, job *CallJob) error {
	// perform trimming to maintain size
	// default stream-node-max-entries is 100, so once entries exceed this limit
	// subsequent add will trim to max len.
	// in other words, need to have ~90 pending call jobs before trimming corrupts the job queue.
	// which should be impossible in this case.
	return databases.RedisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: cq.stream,
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

func (cq *CallQueue) GetCallJobs(ctx context.Context) ([]CallJob, error) {
	entries, err := databases.RedisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    cq.worker,
		Consumer: fmt.Sprintf("%d", time.Now().UnixMilli()),
		Streams:  []string{cq.stream, ">"},
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

func (cq *CallQueue) Done(ctx context.Context, job *CallJob) error {
	return databases.RedisClient.XAck(ctx, cq.stream, cq.worker, job.ID).Err()
}

// call logs
func (cq *CallQueue) Log(ctx context.Context, job *CallJob) error {
	logKey := getLogKey(job.RoomID)

	jobstr, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return databases.RedisClient.RPush(ctx, logKey, jobstr).Err()
}

func (cq *CallQueue) ListLogs(ctx context.Context, roomID string, lastN int64) ([]CallJob, error) {
	logKey := getLogKey(roomID)

	var res *redis.StringSliceCmd
	if lastN <= 0 {
		res = databases.RedisClient.LRange(ctx, logKey, 0, -1)
	} else {
		res = databases.RedisClient.LRange(ctx, logKey, -lastN, -1)
	}

	if res.Err() != nil {
		return nil, res.Err()
	}

	logsstr := res.Val()

	var jobs []CallJob
	for _, logstr := range logsstr {
		var job CallJob
		if err := json.Unmarshal([]byte(logstr), &job); err != nil {
			logs.Error("failed to unmarshal call log: %s", err.Error())
			continue
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func getLogKey(roomID string) string {
	return fmt.Sprintf("call_log:%s:%s", roomID, time.Now().Format("20060102"))
}
