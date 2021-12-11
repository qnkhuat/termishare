package main

import (
	"github.com/qnkhuat/termishare/pkg/logging"
	"github.com/qnkhuat/termishare/pkg/termishare"
)

func main() {
	logging.Config(".log", "TERMISHARE: ")

	ts := termishare.New()
	ts.Start()
	return
}
