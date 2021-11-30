package main

import (
	util "github.com/qnkhuat/termishare/internal/util"
	pty "github.com/qnkhuat/termishare/pkg/pty"
	"io"
	"os"
)

func main() {
	tty := pty.New()
	tty.StartShell([]string{})
	tty.MakeRaw()

	// Pipe command's response to tty
	go func() {
		_, err := io.Copy(os.Stdout, tty.F())
		util.Chk(err, "Failed to send tty's output to std")
	}()

	// Pipe what user type to terminal session
	go func() {
		_, err := io.Copy(tty.F(), os.Stdin)
		util.Chk(err, "Failed to send stdin to pty")
	}()

	// blocking
	tty.Wait()
}
