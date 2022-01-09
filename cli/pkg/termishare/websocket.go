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
	*websocket.Conn
	In             chan message.Wrapper
	Out            chan message.Wrapper
	lastActiveTime time.Time
	active         bool
}

func NewWebSocketConnection(url string) (*WebSocket, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)

	if err != nil {
		return nil, err
	}

	return &WebSocket{
		Conn:   conn,
		In:     make(chan message.Wrapper, cfg.TERMISHARE_WEBSOCKET_CHANNEL_SIZE),
		Out:    make(chan message.Wrapper, cfg.TERMISHARE_WEBSOCKET_CHANNEL_SIZE),
		active: true,
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
				err := ws.WriteJSON(msg)
				if err != nil {
					log.Printf("Failed to send mesage : %s", err)
					ws.Stop()
					break
				}
			} else {
				log.Printf("Failed to get message from channel")
				ws.Stop()
				break
			}
		}
	}()

	// Send message coroutine
	for {
		msg := message.Wrapper{}
		err := ws.ReadJSON(&msg)
		if err == nil {
			ws.In <- msg // Will be handled in Room
		} else {
			log.Printf("Failed to read message. Closing connection: %s", err)
			ws.Stop()
			break
		}
	}
	log.Printf("Out websocket")
}

// Gracefully close websocket connection
func (ws *WebSocket) Stop() {
	if ws.active {
		ws.active = false
		log.Printf("Closing client")
		ws.WriteControl(websocket.CloseMessage, []byte{}, time.Time{})
		time.Sleep(1 * time.Second) // give client sometimes to receive the control message
		close(ws.In)
		close(ws.Out)
		ws.Close()
	}
}
