package models

import (
	"encoding/json"
	"io"

	"git.rpjosh.de/RPJosh/go-logger"
)

// For every request a response message for the client
// is returned that contains a small phrase describing
// the operation and its status
type ResponseMessage struct {

	// The response message in the client language
	Client string `json:"client"`
}

// ResponseMessageWrapper is a wrapper around the struct "ResponseMessage"
// that should be uesed if the message is not included in another entity
type ResponseMessageWrapper struct {
	Message ResponseMessage `json:"message"`
}

// NewResponseMessage decodes the JSON response of the given reader
// to a new message
func NewResponseMessageWrapper(r io.Reader) *ResponseMessageWrapper {
	var msg ResponseMessageWrapper

	if err := json.NewDecoder(r).Decode(&msg); err != nil {
		logger.Warning("Failed to decode response message: %s", err)
	}

	return &msg
}

func (m ResponseMessage) String() string {
	return m.Client
}

func (m ResponseMessageWrapper) String() string {
	return m.Message.Client
}
