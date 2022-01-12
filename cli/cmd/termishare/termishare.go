package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/qnkhuat/termishare/internal/cfg"
	"github.com/qnkhuat/termishare/pkg/logging"
	"github.com/qnkhuat/termishare/pkg/termishare"
)

func main() {
	var server = flag.String("server", "https://termishare.com", "Address to signaling server")
	var noTurn = flag.Bool("no-turn", false, "Don't use a TURN server")
	flag.Parse()
	args := flag.Args()

	// if termishare get an argument that are not a flag, use it as the client
	if len(args) == 1 {
		logging.Config("/tmp/termishare.log", "REMOTE CLIENT: ")
		// use as a remote client

		rc := termishare.NewRemoteClient()

		re := regexp.MustCompile(`^((http|https):\/\/[^\s/]+)\/([^\s/]+)*`)
		matches := re.FindSubmatch([]byte(args[0]))
		if len(matches) == 4 {
			// url with template http://server.com/sessionID
			rc.Connect(string(matches[1]), string(matches[3]))
		} else if !strings.Contains(args[0], "/") {
			// guessing we're passed with only sessionID
			rc.Connect(*server, args[0])
		}
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
