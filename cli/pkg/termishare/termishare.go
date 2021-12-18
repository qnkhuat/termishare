package termishare

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	ptyDevice "github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/qnkhuat/termishare/internal/cfg"
	"github.com/qnkhuat/termishare/pkg/message"
	"github.com/qnkhuat/termishare/pkg/pty"
)

type Client struct {
	// for transferring terminal changes
	termishareChannel *webrtc.DataChannel

	// for transferring config like winsize
	configChannel *webrtc.DataChannel

	conn *webrtc.PeerConnection
}

type Termishare struct {
	pty *pty.Pty

	// Used for singnaling
	wsConn *websocket.Conn

	clients map[string]*Client
	lock    sync.RWMutex
}

func New() *Termishare {
	return &Termishare{
		pty:     pty.New(),
		clients: make(map[string]*Client),
	}
}

func (ts *Termishare) Start() error {
	// Create a pty to fake the terminal session
	// TODO: make it have sessionid
	envVars := []string{fmt.Sprintf("%s=%s", cfg.TERMISHARE_ENVKEY_SESSIONID, "ngockq")}
	ts.pty.StartShell(envVars)
	fmt.Printf("Press Enter to continue!")
	bufio.NewReader(os.Stdin).ReadString('\n')
	ts.pty.MakeRaw()
	defer ts.Stop("Bye!")

	// Initiate websocket connection for signaling
	wsConn, _, err := websocket.DefaultDialer.Dial("ws://localhost:3000/ws", nil)
	//wsConn, _, err := websocket.DefaultDialer.Dial("wss://server.termishare.com/ws", nil)
	if err != nil {
		fmt.Printf("Failed to connect to websocket server: %s", err)
		ts.Stop("Failed to connect to websocket server")
		return err
	}
	ts.wsConn = wsConn

	wsConn.SetCloseHandler(func(code int, text string) error {
		log.Printf("WebSocket connection closed with code %d :%s", code, text)
		return nil
	})

	go ts.startHandleWsMessages()

	// Send a winsize message when ever terminal change size
	ts.pty.SetWinChangeCB(func(ws *ptyDevice.Winsize) {
		ts.broadcastConfig(message.Wrapper{
			Type: message.TTermWinsize,
			Data: message.Winsize{
				Rows: ws.Rows,
				Cols: ws.Cols},
		})
	})

	// Pipe command response to Pty and server
	go func() {
		// Write both to stdout and remote
		mw := io.MultiWriter(os.Stdout, ts)
		_, err := io.Copy(mw, ts.pty.F())
		if err != nil {
			log.Printf("Failed to send pty to mw: %s", err)
			ts.Stop("Failed to connect pty with server\n")
		}
	}()

	// Pipe what user type to terminal session
	go func() {
		_, err := io.Copy(ts.pty.F(), os.Stdin)
		if err != nil {
			log.Printf("Failed to send stdin to pty: %s", err)
			ts.Stop("Failed to get user input\n")
		}
	}()

	ts.pty.Wait() // Blocking until user exit
	return nil
}

func (ts *Termishare) Stop(msg string) {
	if ts.wsConn != nil {
		ts.wsConn.WriteControl(websocket.CloseMessage, []byte{}, time.Time{})
		ts.wsConn.Close()
	}

	if ts.pty != nil {
		ts.pty.Stop()
		ts.pty.Restore()
	}

	fmt.Println(msg)
}

// ------------------------------ WebSocket ------------------------------

func (ts *Termishare) ConnectWs(url string) error {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		ts.Stop(fmt.Sprintf("Failed to connect to websocket server: %s", err))
		return err
	}
	ts.wsConn = conn
	return nil
}

// Blocking call to connect to a websocket server for signaling
func (ts *Termishare) startHandleWsMessages() error {
	if ts.wsConn == nil {
		log.Printf("Websocket connection not initialized")
		return fmt.Errorf("Websocket connection not initialized")
	}

	for {
		msg := message.Wrapper{}
		err := ts.wsConn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Failed to read websocket message: %s", err)
			return err
		}
		log.Printf("Got a message: %v", msg)

		// skip messages that are not send to the host
		if msg.To != cfg.TERMISHARE_WEBSOCKET_HOST_ID {
			log.Printf("Skip message :%s", msg)
			continue
		}

		err = ts.handleWebSocketMessage(msg)
		if err != nil {
			log.Printf("Failed to handle message: %v, with error: %s", msg, err)
			return err
		}

	}

}

func (ts *Termishare) handleWebSocketMessage(msg message.Wrapper) error {

	switch msgType := msg.Type; msgType {
	// offer
	case message.TRTCWillYouMarryMe:
		client, err := ts.newClient(msg.From)
		log.Printf("New client with ID: %s", msg.From)
		if err != nil {
			return fmt.Errorf("Failed to create client: %s", err)
		}

		offer := webrtc.SessionDescription{}
		if err := json.Unmarshal([]byte(msg.Data.(string)), &offer); err != nil {
			return err
		}

		if err := client.conn.SetRemoteDescription(offer); err != nil {
			return fmt.Errorf("Failed to set remote description: %s", err)
		}

		// send back SDP answer and set it as local description
		answer, err := client.conn.CreateAnswer(nil)
		if err != nil {
			return fmt.Errorf("Failed to create offfer: %s", err)
		}

		if err := client.conn.SetLocalDescription(answer); err != nil {
			return fmt.Errorf("Failed to set local description: %s", err)
		}

		answerByte, _ := json.Marshal(answer)
		payload := message.Wrapper{
			Type: message.TRTCYes,
			Data: string(answerByte),
			To:   msg.From,
		}
		ts.writeWebsocket(payload)

	case message.TRTCKiss:
		client, ok := ts.clients[msg.From]
		if !ok {
			return fmt.Errorf("Client with ID: %s not found", msg.From)
		}

		candidate := webrtc.ICECandidateInit{}
		if err := json.Unmarshal([]byte(msg.Data.(string)), &candidate); err != nil {
			return fmt.Errorf("Failed to unmarshall icecandidate: %s", err)
		}

		if err := client.conn.AddICECandidate(candidate); err != nil {
			return fmt.Errorf("Failed to add ice candidate: %s", err)
		}

	default:
		return fmt.Errorf("Not implemented to handle message type: %s", msg.Type)

	}
	return nil
}

// shortcut to write to websocket connection
func (ts *Termishare) writeWebsocket(msg message.Wrapper) error {
	msg.From = cfg.TERMISHARE_WEBSOCKET_HOST_ID
	if ts.wsConn == nil {
		return fmt.Errorf("Websocket not connected")
	}
	err := ts.wsConn.WriteJSON(msg)
	if err != nil {
		log.Printf("Failed to boardcast to websocket connection: %s", err)
		return err
	}
	return nil
}

func (ts *Termishare) broadcastConfig(msg message.Wrapper) error {
	msg.From = cfg.TERMISHARE_WEBRTC_DATA_CHANNEL
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal config: %s", err)
		return err
	}

	ts.lock.RLock()
	defer ts.lock.RUnlock()
	for ID, client := range ts.clients {
		//go func(ID string, client *Client) {
		if client.configChannel != nil {
			err = client.configChannel.Send(payload)
			if err != nil {
				log.Printf("Failed to send config to client: %s :%s", ID, err)
			}
		}
		//}(ID, client)
	}
	return nil
}

// Write method to forward terminal changes over webrtc
func (ts *Termishare) Write(data []byte) (int, error) {
	ts.lock.RLock()
	defer ts.lock.RUnlock()

	for ID, client := range ts.clients {
		//go func(ID string, client *Client) {
		if client.termishareChannel != nil {
			err := client.termishareChannel.Send(data)
			if err != nil {
				log.Printf("Failed to send config to client: %s", ID)
			}
		}
		//}(ID, client)
	}

	return len(data), nil
}
func (ts *Termishare) removeClient(ID string) {
	if client, ok := ts.clients[ID]; ok {
		ts.lock.Lock()
		defer ts.lock.Unlock()
		if client.configChannel != nil {
			client.configChannel.Close()
			client.configChannel = nil
		}

		if client.termishareChannel != nil {
			client.termishareChannel.Close()
			client.termishareChannel = nil
		}

		if client.conn != nil {
			client.conn.Close()
		}

		delete(ts.clients, ID)
	}
}

func (ts *Termishare) newClient(ID string) (*Client, error) {
	// Initiate peer connection
	var config = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{
			URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	client := &Client{}

	ts.lock.Lock()
	ts.clients[ID] = client
	ts.lock.Unlock()

	peerConn, err := webrtc.NewPeerConnection(config)
	if err != nil {
		fmt.Printf("Failed to create peer connection: %s", err)
		return nil, err
	}
	client.conn = peerConn

	peerConn.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("Peer connection state has changed: %s", s.String())
		switch s {
		case webrtc.PeerConnectionStateConnected:
			//time.AfterFunc(500*time.Millisecond,
			go func() {
				ts.pty.Refresh()
				ws, err := pty.GetWinsize(0)
				if err != nil {
					log.Printf("Failed to get winsize after refresh: %s", err)
					return
				}

				// retry send winsize message until client get it
				for {
					msg := message.Wrapper{
						To:   ID,
						Type: message.TTermWinsize,
						Data: message.Winsize{
							Rows: ws.Rows,
							Cols: ws.Cols}}

					payload, err := json.Marshal(msg)
					client := ts.clients[ID]
					if client.configChannel != nil {
						err = client.configChannel.Send(payload)
						if err == nil {
							break
						}
						log.Printf("Failed to send config: %s", err)
					} else {
						log.Printf("Config channel not found, retrying")
						time.Sleep(100 * time.Millisecond)
					}
				}
			}()

		case webrtc.PeerConnectionStateClosed, webrtc.PeerConnectionStateDisconnected:
			log.Printf("Removing client: %s", ID)
			ts.removeClient(ID)
		}
	})

	peerConn.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			switch label := d.Label(); label {

			case cfg.TERMISHARE_WEBRTC_DATA_CHANNEL:
				d.OnMessage(func(msg webrtc.DataChannelMessage) {
					ts.pty.Write(msg.Data)
				})
				ts.clients[ID].termishareChannel = d

			case cfg.TERMISHARE_WEBRTC_CONFIG_CHANNEL:
				d.OnMessage(func(msg webrtc.DataChannelMessage) {
					log.Printf("config channel got message: %v", msg)
				})
				ts.clients[ID].configChannel = d

			default:
				log.Printf("Unhandled data channel with label: %s", d.Label())
			}
		})
	})

	peerConn.OnICECandidate(func(ice *webrtc.ICECandidate) {
		if ice == nil {
			return
		}

		candidate, err := json.Marshal(ice.ToJSON())
		if err != nil {
			log.Printf("Failed to decode ice candidate: %s", err)
			return
		}

		msg := message.Wrapper{
			Type: message.TRTCKiss,
			Data: string(candidate),
			To:   ID,
		}

		ts.writeWebsocket(msg)
	})

	return client, nil
}

func (ts *Termishare) getClient(ID string) *Client {
	ts.lock.RLock()
	defer ts.lock.Unlock()
	return ts.clients[ID]
}
