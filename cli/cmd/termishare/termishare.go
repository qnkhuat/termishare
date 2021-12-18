package main

import (
	"flag"

	"github.com/qnkhuat/termishare/pkg/logging"
	"github.com/qnkhuat/termishare/pkg/termishare"
)

func main() {
	var server = flag.String("server", "localhost:3000", "Address to signalling server")
	flag.Parse()

	logging.Config("/tmp/termishare.log", "TERMISHARE: ")
	ts := termishare.New()
	ts.Start(*server)
	return
}
