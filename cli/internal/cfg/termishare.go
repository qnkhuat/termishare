package cfg

import (
	"github.com/pion/webrtc/v3"
)

const (
	TERMISHARE_WEBSOCKET_CHANNEL_SIZE = 256 // termishare channel buffer size for websocket

	TERMISHARE_ENVKEY_SESSIONID      = "TERMISHARE_SESSIONID" // name of env var to keep sessionid value
	TERMISHARE_WEBRTC_DATA_CHANNEL   = "termishare"           // lable name of webrtc data channel to exchange byte data
	TERMISHARE_WEBRTC_CONFIG_CHANNEL = "config"               // lable name of webrtc config channel to exchange config
	TERMISHARE_WEBSOCKET_HOST_ID     = "host"                 // ID of message sent by the host
)

var TERMISHARE_ICE_SERVER_STUNS = []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302", "stun:stun1.l.google.com:19302"}}}

var TERMISHARE_ICE_SERVER_TURNS = []webrtc.ICEServer{{URLs: []string{"turn:104.237.1.191:3478"},
	Username:   "termishare",
	Credential: "termishareisfun"}}
