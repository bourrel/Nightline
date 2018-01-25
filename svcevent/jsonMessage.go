package svcevent

type jsonBody interface{}

type jsonMessage struct {
	Body jsonBody `json:"body"`
	Name string   `json:"name"`
	User int64    `json:"user"`
}
