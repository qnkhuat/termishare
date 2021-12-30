package message

import (
	"encoding/json"
)

type MType string

const (
	TRTCWillYouMarryMe MType = "WillYouMarryMe" // Offer
	TRTCYes            MType = "Yes"            // Answer
	TRTCKiss           MType = "Kiss"           // Candidate

	TTermWinsize MType = "Winsize" // Update winsize

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
