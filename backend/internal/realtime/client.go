package realtime

import (
	"sync"

	"github.com/gorilla/websocket"
)

const sendBufferSize = 256

type Client struct {
	conn *websocket.Conn
	send chan Event
	once sync.Once
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn: conn,
		send: make(chan Event, sendBufferSize),
	}
}

func (c *Client) WritePump() {
	defer c.conn.Close()

	for event := range c.send {
		if err := c.conn.WriteJSON(event); err != nil {
			return
		}
	}
}

func (c *Client) Close() {
	c.once.Do(func() {
		close(c.send)
	})
}
