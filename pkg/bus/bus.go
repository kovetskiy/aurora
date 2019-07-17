package bus

// Need to have const list of message types

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}
