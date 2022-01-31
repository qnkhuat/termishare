// A RemoteClient that connect to a termishare session from terminal
// Not to confuse it with Client which is a connection between "Client" with Termishare
package termishare

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	ptyDevice "github.com/creack/pty"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/qnkhuat/termishare/internal/cfg"
	"github.com/qnkhuat/termishare/pkg/message"
	"github.com/qnkhuat/termishare/pkg/pty"
)

type RemoteClient struct {
	clientID string

	// use Client struct
	// for transferring terminal changes
	dataChannel *webrtc.DataChannel
	// for transferring config like winsize
	configChannel *webrtc.DataChannel
	peerConn      *webrtc.PeerConnection

	wsConn *WebSocket
	pty    *pty.Pty

	// store the previously pressed key to detect exit sequence
	previousKey byte
	done        chan bool
	winSizes    struct {
		remoteRows uint16
		remoteCols uint16
		thisRows   uint16
		thisCols   uint16
	}
	muteDisplay bool
}

func NewRemoteClient() *RemoteClient {
	return &RemoteClient{
		pty:         pty.New(),
		clientID:    uuid.NewString(),
		done:        make(chan bool),
		muteDisplay: false,
	}
}

func (rc *RemoteClient) Connect(server string, sessionID string) {
	wsURL := GetWSURL(server, sessionID)
	fmt.Printf("Connecting to : %s\n", wsURL)
	fmt.Println("Press 'Ctrl-x + Ctrl-x' to exit")

	fmt.Printf("Press Enter to continue!\n")
	bufio.NewReader(os.Stdin).ReadString('\n')

	wsConn, err := NewWebSocketConnection(wsURL)
	if err != nil {
		log.Printf("Failed to connect to singaling server: %s", err)
		rc.Stop("Failed to connect to signaling server")
	}
	go wsConn.Start()
	rc.wsConn = wsConn

	// will stop stdin from piping to stdout
	rc.pty.MakeRaw()
	defer rc.pty.Restore()

	winsize, err := pty.GetWinsize(0)
	if err != nil {
		rc.Stop("Failed to start")
	}
	rc.winSizes.thisCols = winsize.Cols
	rc.winSizes.thisRows = winsize.Rows

	rc.pty.SetWinChangeCB(func(ws *ptyDevice.Winsize) {
		rc.winSizes.thisCols = ws.Cols
		rc.winSizes.thisRows = ws.Rows
		rc.maybeNeedResize()
	})

	// Initiate peer connection
	iceServers := cfg.TERMISHARE_ICE_SERVER_STUNS
	iceServers = append(iceServers, cfg.TERMISHARE_ICE_SERVER_TURNS...)

	config := webrtc.Configuration{
		ICEServers:   iceServers,
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
	}

	peerConn, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Printf("Failed to create peer connetion : %s", err)
	}

	rc.peerConn = peerConn

	peerConn.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("Peer connection state has changed: %s", s.String())
		switch s {
		case webrtc.PeerConnectionStateClosed:
		case webrtc.PeerConnectionStateDisconnected:
		case webrtc.PeerConnectionStateFailed:
			rc.Stop("Disconnected!")
		}
	})

	configChannel, err := peerConn.CreateDataChannel(cfg.TERMISHARE_WEBRTC_CONFIG_CHANNEL, nil)
	dataChannel, err := peerConn.CreateDataChannel(cfg.TERMISHARE_WEBRTC_DATA_CHANNEL, nil)
	rc.configChannel = configChannel
	rc.dataChannel = dataChannel

	configChannel.OnMessage(func(webrtcMsg webrtc.DataChannelMessage) {
		msg := &message.Wrapper{}
		err := json.Unmarshal(webrtcMsg.Data, msg)
		if err != nil {
			log.Printf("Failed to read config message: %s", err)
			return
		}

		log.Printf("Config channel got msg: %v", msg)
		switch msg.Type {
		case message.TTermWinsize:
			ws := &message.Winsize{}
			err = message.ToStruct(msg.Data, ws)
			if err != nil {
				log.Printf("Failed to decode winsize message: %s", err)
				return
			}

			rc.winSizes.remoteCols = ws.Cols
			rc.winSizes.remoteRows = ws.Rows
			rc.maybeNeedResize()

		default:
			log.Printf("Unhandled msg config type: %s", msg.Type)
		}
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		if !rc.muteDisplay {
			os.Stdout.Write(msg.Data)
		}
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
			Type: message.TRTCCandidate,
			Data: string(candidate),
		}

		rc.sendWebsocket(msg)
	})

	// send offer
	offer, err := peerConn.CreateOffer(nil)
	if err != nil {
		log.Printf("Failed to create offer :%s", err)
		rc.Stop("Failed to connect to termishare session")
	}

	err = peerConn.SetLocalDescription(offer)
	if err != nil {
		log.Printf("Failed to set local description: %s", err)
		rc.Stop("Failed to connect to termishare session")
	}

	offerByte, _ := json.Marshal(offer)
	payload := message.Wrapper{
		Type: message.TRTCOffer,
		Data: string(offerByte),
	}

	rc.sendWebsocket(payload)

	// Read from stdin and send to the host
	stdinReader := bufio.NewReaderSize(os.Stdin, 1)
	go func() {
		for {
			d, err := stdinReader.ReadByte()
			if err != nil {
				log.Printf("Failed to read from stdin: %s", err)
				continue
			}
			// if detects Ctrl-x + Ctrl-x => stop
			if rc.previousKey == byte('\x18') && d == byte('\x18') {
				log.Printf("Escape key detected. Exiting")
				rc.Stop("Disconnected!")
				break
			}
			rc.previousKey = d
			if rc.dataChannel != nil {
				rc.dataChannel.Send([]byte{d})
			}
		}
	}()

	// handle websocket messages
	go func() {
		for {
			msg, ok := <-rc.wsConn.In
			if !ok {
				log.Printf("Failed to read websocket message")
				break
			}

			// only read message sent from the host
			if msg.From != cfg.TERMISHARE_WEBSOCKET_HOST_ID {
				log.Printf("Skip message :%v", msg)
			}

			err := rc.handleWebSocketMessage(msg)
			if err != nil {
				log.Printf("Failed to handle message: %v, with error: %s", msg, err)
				break
			}
		}
	}()

	// Wait
	<-rc.done
	return
}

func (rc *RemoteClient) handleWebSocketMessage(msg message.Wrapper) error {
	switch msgType := msg.Type; msgType {
	// offer
	case message.TRTCOffer:
		return fmt.Errorf("Remote client shouldn't receive Offer message")

	case message.TRTCAnswer:
		answer := webrtc.SessionDescription{}
		if err := json.Unmarshal([]byte(msg.Data.(string)), &answer); err != nil {
			return err
		}

		rc.peerConn.SetRemoteDescription(answer)

	case message.TRTCCandidate:
		candidate := webrtc.ICECandidateInit{}
		if err := json.Unmarshal([]byte(msg.Data.(string)), &candidate); err != nil {
			return fmt.Errorf("Failed to unmarshall icecandidate: %s", err)
		}

		if err := rc.peerConn.AddICECandidate(candidate); err != nil {
			return fmt.Errorf("Failed to add ice candidate: %s", err)
		}

	case message.TWSPing:
		return nil

	default:
		log.Printf("Unhandled message type: %s", msgType)
		return nil
	}

	return nil
}

func (rc *RemoteClient) Stop(msg string) {
	log.Printf("Stop: %s", msg)

	if rc.wsConn != nil {
		rc.wsConn.WriteControl(websocket.CloseMessage, []byte{}, time.Time{})
		rc.wsConn.Close()
		rc.wsConn = nil
	}

	if rc.peerConn != nil {
		rc.peerConn.Close()
		rc.peerConn = nil
	}

	if rc.pty != nil {
		rc.pty.Restore()
		rc.pty = nil
	}

	clearScreen()
	fmt.Println(msg)
	rc.done <- true
	return
}

func (rc *RemoteClient) maybeNeedResize() {
	if (rc.winSizes.remoteCols == 0 && rc.winSizes.remoteRows == 0) || (rc.winSizes.thisCols == 0 && rc.winSizes.thisRows == 0) {
		// not iniated
		return
	}

	if rc.winSizes.thisRows < rc.winSizes.remoteRows || rc.winSizes.thisCols < rc.winSizes.remoteCols {
		rc.muteDisplay = true
		clearScreen()
		fmt.Printf("\n\rYour terminal is smaller than the host's terminal\n\r"+
			"Please resize or press 'Ctrl-x + Ctrl-x' to exit\n\rHost's terminal: %dx%d\n\rYour terminal: %dx%d\n\r",
			rc.winSizes.remoteCols, rc.winSizes.remoteRows, rc.winSizes.thisCols, rc.winSizes.thisRows)
	} else {
		rc.muteDisplay = false
		clearScreen()
		err := rc.requestRemoteRefresh()
		if err != nil {
			log.Printf("Failed to request refresh: %s", err)
		}
	}
}

func (rc *RemoteClient) requestRemoteRefresh() error {
	return rc.sendConfig(message.Wrapper{Type: message.TTermRefresh})
}

func (rc *RemoteClient) sendConfig(msg message.Wrapper) error {
	if rc.configChannel != nil {
		payload, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		return rc.configChannel.Send(payload)
	} else {
		return fmt.Errorf("Config channel not existed")
	}
}

func (rc *RemoteClient) sendWebsocket(msg message.Wrapper) error {
	msg.To = cfg.TERMISHARE_WEBSOCKET_HOST_ID
	msg.From = rc.clientID
	if rc.wsConn == nil {
		return fmt.Errorf("Websocket not connected")
	}
	rc.wsConn.Out <- msg
	return nil
}

func clearScreen() {
	fmt.Fprintf(os.Stdout, "\033[H\033[2J")
}
