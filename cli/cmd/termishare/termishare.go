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
	var server = flag.String("server", "localhost:3000", "Address to signalling server")
	flag.Parse()

	logging.Config("/tmp/termishare.log", "TERMISHARE: ")

	sessionID := os.Getenv(cfg.TERMISHARE_ENVKEY_SESSIONID)
	if sessionID != "" {
		fmt.Printf("This terminal is already being shared at: %s\n", termishare.GetClientURL(sessionID))
	} else {
		ts := termishare.New()
		ts.Start(*server)
	}
	return
}
