package realtime

import "encoding/json"

const EventMessageCreated = "message.created"
const EventMessageDeleted = "message.deleted"
const EventMessageUpdated = "message.updated"
const EventTypingStarted = "typing.started"
const EventTypingStopped = "typing.stopped"

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func (e Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func UnmarshalEvent(payload string) (Event, error) {
	var event Event
	err := json.Unmarshal([]byte(payload), &event)
	return event, err
}
