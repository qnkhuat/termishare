package message

import (
	"encoding/json"
)

type MType string

const (
	TRTCOffer     MType = "Offer"
	TRTCAnswer    MType = "Answer"
	TRTCCandidate MType = "Candidate"

	TTermWinsize MType = "Winsize" // Update winsize

	// Client can order the host to refresh the terminal
	// Used in case client resize and need to update the content to display correctly
	TTermRefresh MType = "Refresh"

	TWSPing MType = "Ping"
)

type Wrapper struct {
	Type MType
	Data interface{}
	From string
	To   string
}

type Winsize struct {
	Rows uint16
	Cols uint16
}

// *** Helper functions ***
func Unwrap(buff []byte) (Wrapper, error) {
	obj := Wrapper{}
	err := json.Unmarshal(buff, &obj)
	return obj, err
}

func Wrap(msgType MType, data string) Wrapper {
	msg := Wrapper{
		Type: msgType,
		Data: data,
	}
	return msg
}

// convert a map to struct
// data is a map
// v is a reference to a typed variable
func ToStruct(data interface{}, v interface{}) error {
	dataByte, _ := json.Marshal(data)
	err := json.Unmarshal(dataByte, v)
	return err
}
