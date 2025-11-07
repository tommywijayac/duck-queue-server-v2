package services

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/beego/beego/v2/core/logs"
	"github.com/tommywijayac/duck-queue-server-v2/backend/models"
)

type EventHubService struct {
	mu   sync.Mutex
	hubs map[string]*hub // room id -> hub
}

type hub struct {
	id         string
	broadcast  chan string
	register   chan *models.Client
	unregister chan *models.Client
	clients    map[*models.Client]struct{}
}

func NewEventHubService() *EventHubService {
	return &EventHubService{
		hubs: make(map[string]*hub),
	}
}

func (eh *EventHubService) RegisterClient(roomId, clientId string, writer http.ResponseWriter) {
	// aggresive locks, who cares
	eh.mu.Lock()
	defer eh.mu.Unlock()

	hub, ok := eh.hubs[roomId]
	if !ok {
		hub = newHub(roomId)
		go hub.run()

		logs.Info("created new room hub ", roomId)
	}

	hub.register <- models.NewClient(clientId, writer)
	hub.broadcast <- "retry: 5000\n" // set reconnect timing for the session

	eh.hubs[roomId] = hub
}

func (eh *EventHubService) UnregisterClient(roomId, clientId string) {
	// aggresive locks, who cares
	eh.mu.Lock()
	defer eh.mu.Unlock()

	hub, ok := eh.hubs[roomId]
	if !ok {
		logs.Warn("room hub not found ", roomId)
		return
	}

	for c := range hub.clients {
		// don't care whether unregister success or not.
		// remove the hub asap so it can't be used (h.broadcast<-msg will block in a race condition if wait until hub ACK closed),
		if len(hub.clients) == 1 {
			delete(eh.hubs, roomId)
		}

		if c.Id == clientId {
			hub.unregister <- c
		}
	}
}

// hub methods
func newHub(id string) *hub {
	return &hub{
		id:         id,
		broadcast:  make(chan string),
		register:   make(chan *models.Client),
		unregister: make(chan *models.Client),
		clients:    make(map[*models.Client]struct{}),
	}
}

func (h *hub) run() {
	for {
		select {
		case client := <-h.register:
			go client.WritePump()
			h.clients[client] = struct{}{}

			logs.Info(fmt.Sprintf("%s: client %s connected", h.id, client.Id))

		case client := <-h.unregister:
			client.Stop <- struct{}{}
			delete(h.clients, client)

			logs.Info(fmt.Sprintf("%s: client %s disconnected", h.id, client.Id))

			if len(h.clients) == 0 {
				logs.Info(fmt.Sprintf("%s: no client left. stop running..", h.id))
				return
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				client.Send <- message
			}
		}
	}
}
