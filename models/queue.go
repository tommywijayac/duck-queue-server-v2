package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/tommywijayac/duck-queue-server-v2/databases"
)

type Queue struct {
	QueueConfig
	// id is the unique identifier for the queue
	id string
}

type QueueConfig struct {
	// how many queue numbers can be in queue at a time
	MaxQueue int
	// how many digits the queue number has
	NumberLen int
	// whether to pad queue number with leading zeroes
	IsPadZeroes bool
}

var DefaultQueueCfg = QueueConfig{
	MaxQueue:    1000,
	NumberLen:   3,
	IsPadZeroes: true,
}

// stored in redis
type QueueItem struct {
	QueueInfo
	Number string `json:"number"`
}

type QueueInfo struct {
	Name  string `json:"name,omitempty"`
	Phone string `json:"phone,omitempty"`
}

func NewQueue(id string, cfg QueueConfig) *Queue {
	return &Queue{
		QueueConfig: cfg,
		id:          id,
	}
}

// Queue Item methods
func (q *Queue) Create(ctx context.Context, roomID string, info QueueInfo) (QueueItem, error) {
	var err error

	keys := q.getKeys()

	// get sequence and generate queue number
	// there is a small chance of race condition since we execute command separately
	// but let's leave it to the field guys	to deal with it
	err = databases.Redis.Incr(ctx, keys["seq"])
	if err != nil {
		return QueueItem{}, err
	}

	seq, err := databases.Redis.Get(ctx, keys["seq"])
	if err != nil {
		return QueueItem{}, err
	}

	number, err := q.generateQueueNumber(string(seq.([]uint8)))
	if err != nil {
		return QueueItem{}, err
	}
	number = roomID + number

	// store queue info
	if err := q.createInfo(ctx, number, info); err != nil {
		return QueueItem{}, err
	}

	// insert to queue
	res := databases.RedisClient.RPush(ctx, keys["base"], number)
	if res.Err() != nil {
		return QueueItem{}, res.Err()
	}

	// update all keys TTL to 18 hours
	expiration := 18 * time.Hour
	databases.RedisClient.Expire(ctx, keys["base"], expiration)
	databases.RedisClient.Expire(ctx, keys["seq"], expiration)
	databases.RedisClient.Expire(ctx, keys["info"], expiration)

	return QueueItem{
		QueueInfo: info,
		Number:    number,
	}, nil
}

func (q *Queue) Move(ctx context.Context, queueNumber string, destination *Queue) error {
	sourceKeys := q.getKeys()
	destKeys := destination.getKeys()

	// check destination queue length
	destLenRes := databases.RedisClient.LLen(ctx, destKeys["base"])
	if destLenRes.Err() != nil {
		return destLenRes.Err()
	}
	if destLenRes.Val() >= int64(destination.MaxQueue) {
		return errors.New("destination queue is full")
	}

	// check & remove queue number from source queue
	res := databases.RedisClient.LRem(ctx, sourceKeys["base"], 1, queueNumber)
	if res.Err() != nil {
		return res.Err()
	}
	if res.Val() == 0 {
		return errors.New("queue number not found in source queue")
	}

	// insert to destination queue
	res = databases.RedisClient.RPush(ctx, destKeys["base"], queueNumber)
	if res.Err() != nil {
		return res.Err()
	}

	return nil
}

func (q *Queue) getKeys() map[string]string {
	// queue:{queue id}:{YYYYMMDD}
	//
	// example values for room id "A", which has 2 counters, on Jan 1st, 2025
	// - queue:A:counter:1:20250101
	// - queue:A:counter:2:20250101
	// - queue:A:main:20250101
	// - queue:A:skip:20250101
	base := fmt.Sprintf("queue:%s:%s", q.id, time.Now().Format("20060102"))

	return map[string]string{
		"base": base,
		"seq":  base + ":seq",
	}
}

func (q *Queue) generateQueueNumber(sequence string) (string, error) {
	var result string

	if len(sequence) > q.NumberLen-1 {
		return "", errors.New("max daily generated number exceeded")
	}

	if q.IsPadZeroes {
		for i := 0; i < q.NumberLen-len(sequence); i++ {
			result += "0"
		}
		result += sequence

		return result, nil
	} else {
		return sequence, nil
	}
}

func (q *Queue) List(ctx context.Context) ([]QueueItem, error) {
	keys := q.getKeys()

	// get all queue numbers
	res := databases.RedisClient.LRange(ctx, keys["base"], 0, -1)
	if res.Err() != nil {
		return nil, res.Err()
	}
	numbers := res.Val()

	// get all queue info
	if len(numbers) == 0 {
		return []QueueItem{}, nil
	}

	// assemble result
	var items []QueueItem
	for _, number := range numbers {
		info, err := q.getInfo(ctx, number)
		if err != nil {
			logs.Error("failed to unmarshal queue info for queue number %s: %s", number, err.Error())
			continue
		}

		items = append(items, QueueItem{
			QueueInfo: info,
			Number:    number,
		})
	}

	return items, nil
}

// Queue Info methods
func (q *Queue) createInfo(ctx context.Context, queueNumber string, info QueueInfo) error {
	infostr, err := json.Marshal(info)
	if err != nil {
		return err
	}

	return databases.Redis.Put(ctx, getInfoKey(queueNumber), infostr, 18*time.Hour)
}

func (q *Queue) getInfo(ctx context.Context, queueNumber string) (QueueInfo, error) {
	res, err := databases.Redis.Get(ctx, getInfoKey(queueNumber))
	if err != nil {
		return QueueInfo{}, err
	}
	if res == nil {
		return QueueInfo{}, nil
	}

	var infostr []byte

	switch v := res.(type) {
	case []uint8:
		infostr = []byte(v)
	default:
		return QueueInfo{}, fmt.Errorf("unexpected cache value type: %T", res)
	}

	var info QueueInfo
	if err := json.Unmarshal(infostr, &info); err != nil {
		return QueueInfo{}, err
	}

	return info, nil
}

func getInfoKey(queueNumber string) string {
	return fmt.Sprintf("queue:%s:info", queueNumber)
}
