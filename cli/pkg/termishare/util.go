package termishare

import (
	"fmt"
)

func GetClientURL(client string, sessionID string) string {
	return fmt.Sprintf("%s/%s", client, sessionID)
}
