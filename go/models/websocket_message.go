package models

import (
	"strings"

	"git.rpjosh.de/RPJosh/go-logger"
)

// WebSocketMessage is the root entity that contains different types of
// WebSocket messages.
// The type is defined by the "Type" field
type WebSocketMessage struct {

	// The type of this Message
	Type WebSocketMessageType `json:"type"`

	// Is set on "WebSocketTypeUpdate"
	Update Update `json:"update"`

	// Is set on "WebSocketTypeExecResponse".
	// This does contain an entry that should be executed and the response returned
	ExecResponse Entry `json:"exec_response"`

	// Is set on "WebSocketTypeNoDb"
	NoDb []*Entry `json:"no_db"`
}

// WebSocketMessageType defines the message type that was received by the
// WebSocket
type WebSocketMessageType int

const (
	WebSocketTypeUpdate WebSocketMessageType = iota
	WebSocketTypeExecResponse
	WebSocketTypeNoDb
	WebSocketTypeUnknown
)

func (m *WebSocketMessageType) UnmarshalJSON(b []byte) error {
	value := strings.Trim(string(b), `"`)
	switch value {
	case "update":
		*m = WebSocketTypeUpdate
	case "exec_response":
		*m = WebSocketTypeExecResponse
	case "no_db":
		*m = WebSocketTypeNoDb
	default:
		// Don't throw an error because new message types could be added on the fly with
		// newer versions
		*m = WebSocketTypeUnknown
		logger.Warning("Unknown message type from WebSocket received: %q", value)
	}

	return nil
}

func (m WebSocketMessageType) String() string {
	switch m {
	case WebSocketTypeUpdate:
		return "update"
	case WebSocketTypeExecResponse:
		return "exec_response"
	case WebSocketTypeNoDb:
		return "no_db"
	case WebSocketTypeUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}
