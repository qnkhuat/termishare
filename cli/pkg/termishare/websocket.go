// DEPRECATED
package termishare

import (
	"github.com/gorilla/websocket"
	"github.com/qnkhuat/termishare/internal/cfg"
	"github.com/qnkhuat/termishare/pkg/message"
	"log"
	"time"
)

// An extension of websocket with go channels
type WebSocket struct {
	conn           *websocket.Conn
	In             chan message.Wrapper
	Out            chan message.Wrapper
	lastActiveTime time.Time
}

func NewWebSocketConnection(url string) (*WebSocket, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	return &WebSocket{
		conn: conn,
		In:   make(chan message.Wrapper, cfg.TERMISHARE_WEBSOCKET_CHANNEL_SIZE),
		Out:  make(chan message.Wrapper, cfg.TERMISHARE_WEBSOCKET_CHANNEL_SIZE),
	}, nil
}

// blocking method that start receive and send websocket message
func (ws *WebSocket) Start() {
	// Receive message coroutine
	go func() {
		for {
			msg, ok := <-ws.Out
			ws.lastActiveTime = time.Now()
			if ok {
				err := ws.conn.WriteJSON(msg)
				if err != nil {
					log.Printf("Failed to boardcast to. wsosing connection")
					ws.Stop()
					return
				}
			} else {
				log.Printf("Failed to get message from channel")
				ws.Stop()
				return
			}
		}
	}()

	// Send message coroutine
	for {
		msg := message.Wrapper{}
		err := ws.conn.ReadJSON(&msg)
		if err == nil {
			ws.In <- msg // Will be handled in Room
		} else {
			log.Printf("Failed to read message. wsosing connection: %s", err)
			ws.Stop()
			return
		}
	}
}

// Gracefully close websocket connection
func (ws *WebSocket) Stop() {
	log.Printf("Closing client")
	ws.conn.WriteControl(websocket.CloseMessage, []byte{}, time.Time{})
	time.Sleep(1 * time.Second) // give client sometimes to receive the control message
	ws.conn.Close()
}
