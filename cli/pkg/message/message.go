package message

import (
	"encoding/json"
)

type MType string

const (
	TRTCWillYouMarryMe MType = "WillYouMarryMe" // Offer
	TRTCYes            MType = "Yes"            // Answer
	TRTCKiss           MType = "Kiss"           // Candidate
)

type Wrapper struct {
	Type MType
	Data string // should be interface{}
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
