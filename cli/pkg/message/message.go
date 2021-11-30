package message

type MType string

const ()

type Wrapper struct {
	Type MType
	Data interface{}

	// time delay of message to take affect
	// this time is relative with the start time of the parent data block it is sent with
	Delay int64 // milliseconds
}
