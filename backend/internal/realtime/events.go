package realtime

const EventMessageCreated = "message.created"

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}
