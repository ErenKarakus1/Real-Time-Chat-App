package realtime

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	sendBufferSize = 256
	pongTimeout    = 60 * time.Second
	pingInterval   = 30 * time.Second
	writeTimeout   = 10 * time.Second
)

type Client struct {
	conn *websocket.Conn
	send chan Event
	once sync.Once
}

type IncomingEventHandler func(event Event)

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn: conn,
		send: make(chan Event, sendBufferSize),
	}
}

func (c *Client) ReadPump(handle IncomingEventHandler) {
	defer c.conn.Close()

	c.conn.SetReadLimit(1024)
	c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
		return nil
	})

	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		if handle == nil {
			continue
		}

		var event Event
		if err := json.Unmarshal(payload, &event); err != nil {
			continue
		}

		handle(event)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	defer c.conn.Close()

	for {
		select {
		case event, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(event); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) Close() {
	c.once.Do(func() {
		close(c.send)
	})
}
