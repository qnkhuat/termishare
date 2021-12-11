package termishare

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	ptyDevice "github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/qnkhuat/termishare/internal/cfg"
	"github.com/qnkhuat/termishare/internal/util"
	"github.com/qnkhuat/termishare/pkg/message"
	"github.com/qnkhuat/termishare/pkg/pty"
)

type Termishare struct {
	pty *pty.Pty

	// Used for singnaling
	wsConn *websocket.Conn

	// The main connection to exchange data
	peerConn     *webrtc.PeerConnection
	dataChannels map[string]*webrtc.DataChannel
}

func New() *Termishare {
	return &Termishare{
		pty:          pty.New(),
		dataChannels: make(map[string]*webrtc.DataChannel),
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
	if err != nil {
		fmt.Printf("Failed to connect to websocket server: %s", err)
		ts.Stop("Failed to connect to websocket server")
		return err
	}
	ts.wsConn = wsConn

	// Initiate peer connection
	var config = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{
			URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
	peerConn, err := webrtc.NewPeerConnection(config)
	if err != nil {
		fmt.Printf("Failed to create peer connection: %s", err)
		ts.Stop("Failed to create Peer Connection")
		return err
	}
	ts.peerConn = peerConn

	peerConn.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("Peer connection state has changed: %s", s.String())
		if s == webrtc.PeerConnectionStateConnected {
			time.AfterFunc(500*time.Millisecond, func() {

				ts.pty.Refresh()
				ws, err := pty.GetWinsize(0)
				if err != nil {
					log.Printf("Failed to get winsize after refresh: %s", err)
					return
				}

				// retry send winsize message until client get it
				for {
					err = ts.writeConfig(message.Wrapper{
						Type: message.TTermWinsize,
						Data: message.Winsize{
							Rows: ws.Rows,
							Cols: ws.Cols}})
					if err == nil {
						break
					}
					time.Sleep(250 * time.Millisecond)
				}
			})
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
				ts.dataChannels[cfg.TERMISHARE_WEBRTC_DATA_CHANNEL] = d

			case cfg.TERMISHARE_WEBRTC_CONFIG_CHANNEL:
				d.OnMessage(func(msg webrtc.DataChannelMessage) {
					log.Printf("config channel got message: %v", msg)
				})
				ts.dataChannels[cfg.TERMISHARE_WEBRTC_CONFIG_CHANNEL] = d

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
		}

		ts.writeWebsocket(msg)
	})

	go ts.startHandleWsMessages()

	// Send a winsize message when ever terminal change size
	ts.pty.SetWinChangeCB(func(ws *ptyDevice.Winsize) {
		ts.writeConfig(message.Wrapper{
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
		return fmt.Errorf("Websocket connection not initialized")
	}

	for {
		msg := message.Wrapper{}
		err := ts.wsConn.ReadJSON(&msg)
		log.Printf("Received a message: %v", msg)
		if err != nil {
			log.Printf("Failed to read websocket message: %s", err)
			return err
		}
		ts.handleWebSocketMessage(msg)
	}

}

func (ts *Termishare) handleWebSocketMessage(msg message.Wrapper) error {
	switch msgType := msg.Type; msgType {
	// offer
	case message.TRTCWillYouMarryMe:
		offer := webrtc.SessionDescription{}
		if err := json.Unmarshal([]byte(msg.Data.(string)), &offer); err != nil {
			log.Println(err)
			return err
		}

		if err := ts.peerConn.SetRemoteDescription(offer); err != nil {
			log.Printf("Failed to set remote description: %s", err)
			return err
		}

		// send back SDP answer and set it as local description
		answer, err := ts.peerConn.CreateAnswer(nil)
		if err != nil {
			log.Printf("Failed to create Offer")
			return err
		}

		if err := ts.peerConn.SetLocalDescription(answer); err != nil {
			log.Printf("Failed to set local description: %v", err)
			return err
		}

		answerByte, _ := json.Marshal(answer)
		payload := message.Wrapper{
			Type: message.TRTCYes,
			Data: string(answerByte),
		}
		ts.writeWebsocket(payload)

	case message.TRTCKiss:
		candidate := webrtc.ICECandidateInit{}
		if err := json.Unmarshal([]byte(msg.Data.(string)), &candidate); err != nil {
			log.Println(err)
			return err
		}

		if err := ts.peerConn.AddICECandidate(candidate); err != nil {
			log.Println(err)
			return err
		}

	default:
		log.Printf("Not implemented to handle message type: %s", msg.Type)

	}
	return nil
}

// shortcut to write to websocket connection
func (ts *Termishare) writeWebsocket(msg message.Wrapper) error {
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

func (ts *Termishare) writeConfig(msg message.Wrapper) error {
	if channel, ok := ts.dataChannels[cfg.TERMISHARE_WEBRTC_CONFIG_CHANNEL]; ok {
		payload, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal config: %s", err)
			return err
		}
		err = channel.Send(payload)
		util.Chk(err, "Failed to Send config")
		return err
	}
	return fmt.Errorf("Config channel not found")
}

// Write method to forward terminal changes over webrtc
func (ts *Termishare) Write(data []byte) (int, error) {
	if channel, ok := ts.dataChannels[cfg.TERMISHARE_WEBRTC_DATA_CHANNEL]; ok {
		channel.Send(data)
	} else {
		log.Printf("Termishare channel not found")
	}
	return len(data), nil
}
