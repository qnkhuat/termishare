package termishare

import (
	"fmt"
	"net/url"
	"strings"
)

func GetClientURL(client string, sessionID string) string {
	return fmt.Sprintf("%s/%s", client, sessionID)
}

func GetWSURL(server string, sessionID string) string {
	// Initiate websocket connection for signaling
	scheme := "ws"
	if strings.HasPrefix(server, "https") || strings.HasPrefix(server, "wss") {
		scheme = "wss"
	}
	host := strings.Replace(strings.Replace(server, "http://", "", 1), "https://", "", 1)
	url := url.URL{Scheme: scheme, Host: host, Path: fmt.Sprintf("/ws/%s", sessionID)}
	return url.String()
}
