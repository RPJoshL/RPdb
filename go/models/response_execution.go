package models

// Response of an execution for entries with an attribute of the type
// exec_response
type ExecutionResponse struct {
	// The ID of the entry that was executed
	EntryId int `json:"entry_id"`

	// The unix like response code of the execution.
	// A code != 0 indicates an error
	Code int `json:"response_code"`

	// The text message to display for the client
	Text string `json:"response"`
}
