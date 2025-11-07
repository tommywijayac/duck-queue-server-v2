package models

import (
	"fmt"
	"io"
	"net/http"

	"github.com/beego/beego/v2/core/logs"
)

type Client struct {
	Id string

	writer    io.Writer
	writerCtl *http.ResponseController
	Send      chan string
	Stop      chan struct{}
}

func NewClient(id string, writer http.ResponseWriter) *Client {
	return &Client{
		Id:   id,
		Send: make(chan string),
		Stop: make(chan struct{}),

		writer:    writer,
		writerCtl: http.NewResponseController(writer),
	}
}

// WritePump waits until message is received in send channel, then write to the client via writer.
// WritePump should be run in a goroutine.
func (c *Client) WritePump() {
	for {
		select {
		case <-c.Stop:
			return
		case message := <-c.Send:
			if _, err := fmt.Fprint(c.writer, message); err != nil {
				logs.Info("fail to send message to client ", c.Id)
			}
			c.writerCtl.Flush()
		}
	}
}
