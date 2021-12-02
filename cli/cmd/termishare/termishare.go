package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/qnkhuat/termishare/internal/util"
	"github.com/qnkhuat/termishare/pkg/logging"
	"github.com/qnkhuat/termishare/pkg/message"
	"github.com/qnkhuat/termishare/pkg/pty"
	"io"
	"log"
	"os"
	"time"
)

func main() {
	logging.Config(".log", "TERMISHARE")
	tty := pty.New()
	tty.StartShell([]string{})
	tty.MakeRaw()

	// Pipe command's response to tty
	go func() {
		_, err := io.Copy(os.Stdout, tty.F())
		util.Chk(err, "Failed to send tty's output to std")
		stop(tty)
	}()

	// Pipe what user type to terminal session
	go func() {
		_, err := io.Copy(tty.F(), os.Stdin)
		util.Chk(err, "Failed to send stdin to pty")
		stop(tty)
	}()

	wsConn, _, err := websocket.DefaultDialer.Dial("ws://localhost:3000/ws", nil)
	util.Must(err, "Failed to connect websocket")
	connect(wsConn)

	// blocking
	tty.Wait()
	stop(tty)
}

func stop(tty *pty.Pty) {
	tty.Stop()
	tty.Restore()

}

func connect(wsConn *websocket.Conn) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{
			URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	peerConn, err := webrtc.NewPeerConnection(config)
	util.Chk(err, "Failed to create Peer to peer connection")

	peerConn.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			log.Println("Peer Connection has gone to failed exiting")
			os.Exit(0)
		}
	})

	// Register data channel creation handling
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

		// Register text message handling
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
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

		wsConn.WriteJSON(payload)
	})

	go func() {
		for {
			msg := message.Wrapper{}
			err := wsConn.ReadJSON(&msg)
			util.Chk(err, "Failed to read message from websocket")

			switch msgType := msg.Type; msgType {
			case message.TRTCWillYouMarryMe:
				offer := webrtc.SessionDescription{}
				if err := json.Unmarshal([]byte(msg.Data), &offer); err != nil {
					log.Println(err)
					return
				}

				if err := peerConn.SetRemoteDescription(offer); err != nil {
					log.Printf("Failed to set remote description: %s", err)
					return
				}

				// send back SDP answer and set it as local description
				answer, err := peerConn.CreateAnswer(nil)
				if err != nil {
					log.Printf("Failed to create Offer")
					return
				}

				if err := peerConn.SetLocalDescription(answer); err != nil {
					log.Printf("Failed to set local description: %v", err)
					return
				}

				answerByte, _ := json.Marshal(answer)
				payload := message.Wrapper{
					Type: message.TRTCYes,
					Data: string(answerByte),
				}
				if err = wsConn.WriteJSON(payload); err != nil {
					log.Printf("Failed to send answer: %s", err)
				}

			case message.TRTCKiss:
				candidate := webrtc.ICECandidateInit{}
				if err := json.Unmarshal([]byte(msg.Data), &candidate); err != nil {
					log.Println(err)
					return
				}

				if err := peerConn.AddICECandidate(candidate); err != nil {
					log.Println(err)
					return
				}

			default:
				log.Printf("Not implemented to handle message type: %s", msg.Type)

			}

		}
	}()

	// Block forever
	select {}
}
