package termishare

import (
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	pty "github.com/qnkhuat/termishare/pkg/pty"
)

type Termishare struct {
	pty *pty.Pty

	// Used for singnaling
	wsConn *websocket.Conn

	// The main connection to exchange data
	peerConns []*webrtc.PeerConnection
}

func New() *Termishare {

	return &Termishare{}
}

func (ts *Termishare) ConnectRoom() {

}
