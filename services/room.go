package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/tommywijayac/duck-queue-server-v2/backend/models"
)

type RoomService struct {
	rooms map[string]*models.Room
}

func NewRoomService() *RoomService {
	rooms, err := loadRooms()
	if err != nil {
		logs.Critical("failed to create room service: failed to load rooms: %s", err.Error())
		panic(err)
	}
	logs.Info("Room configuration loaded successfully")

	return &RoomService{
		rooms: rooms,
	}
}

func loadRooms() (map[string]*models.Room, error) {
	// read config
	var cfg map[string]models.RoomDetail
	configFile, err := web.AppConfig.String("room::rooms")
	if err != nil {
		configFile = "conf/rooms.json"
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	// load
	rooms := make(map[string]*models.Room)
	for roomId, rdetail := range cfg {
		rooms[roomId] = models.NewRoom(roomId, rdetail)
	}

	return rooms, nil
}

func (rs *RoomService) CreateQueue(ctx context.Context, sourceRoomId, destRoomId string, info models.QueueInfo) (models.QueueItem, error) {
	sourceRoom, exists := rs.rooms[sourceRoomId]
	if !exists {
		return models.QueueItem{}, errors.New("source room not found")
	}
	destRoom, exists := rs.rooms[destRoomId]
	if !exists {
		return models.QueueItem{}, errors.New("destination room not found")
	}

	if !isActionAllowed(models.RoomActionCreate, sourceRoom, destRoom) {
		return models.QueueItem{}, fmt.Errorf("'create' action %s to %s is not allowed", sourceRoom.Id, destRoom.Id)
	}

	return destRoom.CreateQueue(ctx, info)
}

func (rs *RoomService) ProcessQueue(ctx context.Context, roomId, originQueue, counterId, queueNumber string) error {
	room, exists := rs.rooms[roomId]
	if !exists {
		return errors.New("room not found")
	}

	_, exists = room.Counters[counterId]
	if !exists {
		return errors.New("counter not found in room")
	}

	return room.ProcessQueue(ctx, originQueue, counterId, queueNumber)
}

func (rs *RoomService) CallQueue(ctx context.Context, roomId, counterId, queueNumber string) error {
	room, exists := rs.rooms[roomId]
	if !exists {
		return errors.New("room not found")
	}

	_, exists = room.Counters[counterId]
	if !exists {
		return errors.New("counter not found in room")
	}

	if err := room.CallQueue(ctx, counterId, queueNumber); err != nil {
		return err
	}

	/*
		// audio sent is based on audio template identified by room type
		// note: quick hack -- ideally add new config field
		var audioTemplate model.AudioTemplate
		if info.RoomName == "Registration" {
			audioTemplate = model.AudioTemplateFrontline
		} else if info.RoomName == "Pharmacy" {
			audioTemplate = model.AudioTemplateFrontline
		}

		// send audio to all registered audio out room:
		// - room
		// - main_display that register this room as its member
		roomOut := []string{info.RoomID}
		for displayID, cfg := range uc.listGroupConfig {
			for _, member := range cfg.RoomList {
				if info.RoomID == member {
					roomOut = append(roomOut, displayID)
				}
			}
		}
	*/

	// TODO: play audio on device
	/*
		duration, err := uc.rNotifier.SendAudio(ctx, model.NotifierMessage{
			RoomOut:       roomOut,
			AudioTemplate: audioTemplate,
			QueueNumber:   info.Qnum,
			RoomID:        info.RoomID,
			CounterID:     info.CounterID,
		})
		if err != nil {
			log.Warnf("fail to call room %s: %s", info.RoomID, err.Error())
		}
	*/

	// TODO: send visual and audio to device
	logs.Info("Calling queue number %s at room %s counter %s", queueNumber, roomId, counterId)

	// wait until audio is done played
	duration := 5 // TODO: remove, got wait duration from audio duration
	time.Sleep(time.Duration(duration) * time.Second)

	return nil
}

func (rs *RoomService) SkipQueue(ctx context.Context, roomId, counterId, queueNumber string) error {
	room, exists := rs.rooms[roomId]
	if !exists {
		return errors.New("room not found")
	}

	_, exists = room.Counters[counterId]
	if !exists {
		return errors.New("counter not found in room")
	}

	return room.SkipQueue(ctx, counterId, queueNumber)
}

func (rs *RoomService) MoveQueue(ctx context.Context, sourceRoomId, destRoomId, counterId, queueNumber string) error {
	sourceRoom, exists := rs.rooms[sourceRoomId]
	if !exists {
		return errors.New("source room not found")
	}
	destRoom, exists := rs.rooms[destRoomId]
	if !exists {
		return errors.New("destination room not found")
	}

	if !isActionAllowed(models.RoomActionMove, sourceRoom, destRoom) {
		return fmt.Errorf("'move' action %s to %s is not allowed", sourceRoom.Id, destRoom.Id)
	}

	_, exists = sourceRoom.Counters[counterId]
	if !exists {
		return errors.New("counter not found in room")
	}

	return sourceRoom.MoveQueue(ctx, counterId, queueNumber, destRoom)
}

func isActionAllowed(action models.RoomAction, sourceRoom, destRoom *models.Room) bool {
	for _, a := range sourceRoom.Actions {
		if a.Action != action {
			continue
		}

		if slices.Contains(a.DestinationRoomIDs, destRoom.Id) {
			return true
		}
	}
	return false
}

func (rs *RoomService) GetRoom(ctx context.Context, roomId string) (*models.Room, error) {
	room, exists := rs.rooms[roomId]
	if !exists {
		return nil, errors.New("room not found")
	}
	return room, nil
}

func (rs *RoomService) GetRoomQueues(ctx context.Context, roomId string) (map[string][]models.QueueItem, error) {
	room, exists := rs.rooms[roomId]
	if !exists {
		return nil, errors.New("room not found")
	}
	return room.GetQueues(ctx)
}

func (rs *RoomService) GetRoomDetails(ctx context.Context, roomId string) (models.RoomDetail, map[string]models.RoomCounterDetail, error) {
	room, exists := rs.rooms[roomId]
	if !exists {
		return models.RoomDetail{}, nil, errors.New("room not found")
	}

	return room.RoomDetail, room.Counters, nil
}
