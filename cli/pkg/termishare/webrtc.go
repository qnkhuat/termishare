package termishare

import (
	"github.com/pion/webrtc/v3"
	"log"
)

// An extension of WebRTC with go channels
var config = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{{
		URLs: []string{"stun:stun.l.google.com:19302"}},
	},
}

func NewPeerConnection() (*webrtc.PeerConnection, error) {
	peerConn, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	peerConn.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("Peer connection state has changed: %s", s.String())
	})

	return peerConn, nil
}
