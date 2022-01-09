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

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/qnkhuat/termishare/internal/cfg"
	"github.com/qnkhuat/termishare/pkg/message"
	"github.com/qnkhuat/termishare/pkg/pty"
)

type RemoteClient struct {
	sessionID string
	clientID  string
	// for transferring terminal changes
	dataChannel *webrtc.DataChannel

	// for transferring config like winsize
	configChannel *webrtc.DataChannel

	peerConn *webrtc.PeerConnection

	wsConn *WebSocket

	pty *pty.Pty

	// store the previously pressed key to detect exit sequence
	previousKey byte

	done chan bool
}

func NewRemoteClient() *RemoteClient {
	return &RemoteClient{
		pty:      pty.New(),
		clientID: uuid.NewString(),
		done:     make(chan bool),
	}
}

func (rc *RemoteClient) Connect(server string, sessionID string) {
	rc.sessionID = sessionID
	wsURL := GetWSURL(server, rc.sessionID)
	fmt.Printf("Connecting to : %s\n", wsURL)
	fmt.Println("Press 'Ctrl-c + Ctrl-x' to exit")

	wsConn, err := NewWebSocketConnection(wsURL)
	if err != nil {
		log.Printf("Failed to connect to singaling server: %s", err)
		rc.Stop("Failed to connect to signaling server")
	}
	go wsConn.Start()

	// will stop stdin from piping to stdout
	rc.pty.MakeRaw()
	defer rc.pty.Restore()

	rc.wsConn = wsConn

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

	configChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("Config got channel: %v", msg)
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		os.Stdout.Write(msg.Data)
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

		rc.writeWebsocket(msg)
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
		Type: message.TRTCWillYouMarryMe,
		Data: string(offerByte),
	}

	rc.writeWebsocket(payload)

	// Read from stdin and send to the host
	stdinReader := bufio.NewReaderSize(os.Stdin, 1)
	go func() {
		for {
			d, err := stdinReader.ReadByte()
			if err != nil {
				log.Printf("Failed to read from stdin: %s", err)
				continue
			}
			// if detects Ctrl-c + Ctrl-x => stop
			if rc.previousKey == byte('\x03') && d == byte('\x18') {
				log.Printf("Escape key detected. Exiting")
				rc.Stop("Disconnected!")
				break
			}
			rc.previousKey = d
			if rc.dataChannel != nil {
				rc.dataChannel.Send([]byte{d})
			}
		}

		log.Printf("out read from stdin")
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
		log.Printf("out read from websocket")
	}()

	// Wait
	<-rc.done
	return
}

func (rc *RemoteClient) startHandleWsMessages() error {
	return nil
}

func (rc *RemoteClient) handleWebSocketMessage(msg message.Wrapper) error {
	switch msgType := msg.Type; msgType {
	// offer
	case message.TRTCWillYouMarryMe:
		return fmt.Errorf("Remote client shouldn't receive WillYouMarryMe message")

	case message.TRTCYes:
		answer := webrtc.SessionDescription{}
		if err := json.Unmarshal([]byte(msg.Data.(string)), &answer); err != nil {
			return err
		}

		rc.peerConn.SetRemoteDescription(answer)

	case message.TRTCKiss:
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

func (rc *RemoteClient) writeWebsocket(msg message.Wrapper) error {
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
