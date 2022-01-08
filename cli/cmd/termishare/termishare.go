package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/qnkhuat/termishare/internal/cfg"
	"github.com/qnkhuat/termishare/pkg/logging"
	"github.com/qnkhuat/termishare/pkg/termishare"
)

func main() {
	var server = flag.String("server", "https://termishare.com", "Address to signaling server")
	var connect = flag.String("connect", "", "SessionID to connect to ")
	var noTurn = flag.Bool("no-turn", false, "Don't use a TURN server")
	flag.Parse()

	if *connect != "" {
		logging.Config("/tmp/termishare.log", "REMOTE CLIENT: ")
		// use as a remote client
		rc := termishare.NewRemoteClient()
		rc.Connect(*server, *connect)
		return
	} else {
		logging.Config("/tmp/termishare.log", "TERMISHARE: ")
		// use as a host
		sessionID := os.Getenv(cfg.TERMISHARE_ENVKEY_SESSIONID)

		if sessionID != "" {
			fmt.Printf("This terminal is already being shared at: %s\n", termishare.GetClientURL(*server, sessionID))
		} else {
			ts := termishare.New()
			ts.Start(*server, *noTurn)
		}
		return
	}
}
