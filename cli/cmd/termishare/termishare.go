package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/qnkhuat/termishare/internal/cfg"
	"github.com/qnkhuat/termishare/pkg/logging"
	"github.com/qnkhuat/termishare/pkg/termishare"
)

func main() {
	var server = flag.String("server", "https://server.termishare.com", "Address to signaling server")
	var client = flag.String("client", "https://termishare.com", "Termishare web client URL")
	var connect = flag.String("connect", "", "SessionID to connect to ")
	var noTurn = flag.Bool("no-turn", false, "Don't use a TURN server")
	flag.Parse()

	logging.Config("/tmp/termishare.log", "TERMISHARE: ")
	log.Printf("Config : %v", flag.Args())

	if *connect != "" {
		// use as a remote client
		rc := termishare.NewRemoteClient()
		rc.Connect(*server, *connect)

		return
	} else {
		// use as a host
		sessionID := os.Getenv(cfg.TERMISHARE_ENVKEY_SESSIONID)
		if sessionID != "" {
			fmt.Printf("This terminal is already being shared at: %s\n", termishare.GetClientURL(*client, sessionID))
		} else {
			ts := termishare.New()
			ts.Start(*server, *client, *noTurn)
		}
		return
	}
}
