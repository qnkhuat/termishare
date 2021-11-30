package util

import (
	"log"
)

// Check if error then log with a messages
func Chk(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}

// Check if error is nil, if not Panic
func Must(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
