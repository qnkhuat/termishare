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
		// use as a remote client
		logging.Config("/tmp/termishare.log", "REMOTE CLIENT: ")

		rc := termishare.NewRemoteClient()
		// url with template http://server.com/sessionID
		serverURLRe := regexp.MustCompile(`^((http|https):\/\/[^\s/]+)\/([^\s/]+)*`)
		matches := serverURLRe.FindSubmatch([]byte(args[0]))
		if len(matches) == 4 {
			rc.Connect(string(matches[1]), string(matches[3]))
		} else if !strings.Contains(args[0], "/") {
			// Use default server with a sessionID
			rc.Connect(*server, args[0])
		} else {
			fmt.Println("Failed to parse arguments")
		}
		return
	} else {
		// use as a host
		logging.Config("/tmp/termishare.log", "TERMISHARE: ")
		sessionID := os.Getenv(cfg.TERMISHARE_ENVKEY_SESSIONID)

		if sessionID != "" {
			fmt.Printf("This terminal is already being shared at: %s\n", termishare.GetClientURL(*server, sessionID))
			return
		}
		ts := termishare.New(*noTurn)
		ts.Start(*server)
		return
	}
}
