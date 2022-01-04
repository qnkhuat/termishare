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

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/qnkhuat/termishare/internal/cfg"
	"github.com/qnkhuat/termishare/pkg/message"
	"github.com/qnkhuat/termishare/pkg/pty"
)

type RemoteClient struct {
	sessionID string
	// for transferring terminal changes
	termishareChannel *webrtc.DataChannel

	// for transferring config like winsize
	configChannel *webrtc.DataChannel

	peerConn *webrtc.PeerConnection

	wsConn *WebSocket

	pty *pty.Pty
}

func NewRemoteClient() *RemoteClient {
	return &RemoteClient{
		pty: pty.New()}
}

func (rc *RemoteClient) Connect(server string, sessionID string) {
	wsURL := GetWSURL(server, rc.sessionID)
	fmt.Printf("Connecting to : %s\n", wsURL)

	envVars := []string{fmt.Sprintf("%s=%s", cfg.TERMISHARE_ENVKEY_SESSIONID, rc.sessionID)}
	rc.pty.StartShell(envVars)
	fmt.Printf("Press Enter to continue!\n")
	bufio.NewReader(os.Stdin).ReadString('\n')

	rc.pty.MakeRaw()
	defer rc.Stop("Bye!")

	wsConn, err := NewWebSocketConnection(wsURL)
	if err != nil {
		log.Printf("Failed to connect to singaling server: %s", err)
		rc.Stop("Failed to connect to signaling server")
	}

	rc.wsConn = wsConn
	// Initiate peer connection
	ICEServers := cfg.TERMISHARE_ICE_SERVER_STUNS
	ICEServers = append(ICEServers, cfg.TERMISHARE_ICE_SERVER_TURNS...)

	var config = webrtc.Configuration{
		ICEServers: ICEServers,
	}

	peerConn, err := webrtc.NewPeerConnection(config)
	rc.peerConn = peerConn

	peerConn.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("Peer connection state has changed: %s", s.String())
	})

	go rc.startHandleWsMessages()

	// send offer
	offer, err := peerConn.CreateOffer(nil)
	if err != nil {
		log.Printf("Failed to create offer :%s", err)
		rc.Stop("Failed to connect to termishare session")
	}

	offerByte, _ := json.Marshal(offer)
	payload := message.Wrapper{
		Type: message.TRTCYes,
		Data: string(offerByte),
	}
	rc.writeWebsocket(payload)

}

func (rc *RemoteClient) startHandleWsMessages() error {
	for {
		msg, ok := <-rc.wsConn.In
		if !ok {
			log.Printf("Failed to read websocket message")
			return fmt.Errorf("Failed to read message from websocket")
		}

		// only read message sent from the host
		if msg.From != cfg.TERMISHARE_WEBSOCKET_HOST_ID {
			log.Printf("Skip message :%v", msg)
		}

		err := rc.handleWebSocketMessage(msg)
		if err != nil {
			log.Printf("Failed to handle message: %v, with error: %s", msg, err)
			return err
		}
	}
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
	}

	if rc.pty != nil {
		rc.pty.Stop()
		rc.pty.Restore()
	}

	fmt.Println(msg)
}

func (rc *RemoteClient) writeWebsocket(msg message.Wrapper) error {
	msg.To = cfg.TERMISHARE_WEBSOCKET_HOST_ID
	if rc.wsConn == nil {
		return fmt.Errorf("Websocket not connected")
	}
	rc.wsConn.Out <- msg
	return nil
}
