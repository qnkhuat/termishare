package termishare

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/qnkhuat/termishare/internal/util"
	"github.com/qnkhuat/termishare/pkg/message"
	"log"
	"time"
)

// An extension of WebRTC with go channels
var config = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{{
		URLs: []string{"stun:stun.l.google.com:19302"}},
	},
}

func NewPeerConnection(wsConn *websocket.Conn) (*webrtc.PeerConnection, error) {
	peerConn, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	peerConn.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("Peer connection state has changed: %s", s.String())
	})

	// TODO: this should goes to termishare
	peerConn.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			log.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", d.Label(), d.ID())

			for range time.NewTicker(5 * time.Second).C {
				message := "ngockq"

				log.Printf("Sending '%s'\n", message)

				// Send the message as text
				sendErr := d.SendText(message)
				if sendErr != nil {
					panic(sendErr)
				}
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

		payload := message.Wrapper{
			Type: message.TRTCKiss,
			Data: string(candidate),
		}

		err = wsConn.WriteJSON(payload)
		util.Chk(err, "Failed to write to websocket connection")
	})

	return peerConn, nil
}
