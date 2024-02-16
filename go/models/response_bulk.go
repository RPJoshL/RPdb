package models

import (
	"encoding/json"
	"fmt"
	"strings"

	"git.rpjosh.de/RPJosh/go-logger"
)

// BulkResponse is used to return information
// the client needs about a bulk request.
// It contains the number of faild or successfull operations
// and the created or updated objects.
//
// You have to provide the struct or data type of the bulk
// operation
type BulkResponse[T any] struct {

	// A counter of executed operations
	// grupped by their status
	Overview struct {
		Successful int `json:"successful"`
		Errors     int `json:"errors"`
		Exists     int `json:"exists"`
	} `json:"overview"`

	// Short phrase of the operation status (summary) for the client
	Message ResponseMessage `json:"message"`

	// The returned objects of the bulk response
	ResponseData []BulkResponseData[T] `json:"response"`
}

// BulkResponseStatus is a status flag of the operation for a
// single bulk operation data unit.
type BulkResponseStatus int

const (
	StatusFailed BulkResponseStatus = iota
	StatusCreated
	StatusExists
	StatusDeleted
	StatusUpdated
	StatusEqual
)

func (r BulkResponseStatus) String() string {
	strVal := "unknown"
	switch r {
	case StatusCreated:
		strVal = "created"
	case StatusFailed:
		strVal = "failed"
	case StatusExists:
		strVal = "exists"
	case StatusDeleted:
		strVal = "deleted"
	case StatusUpdated:
		strVal = "updated"
	case StatusEqual:
		strVal = "equal"
	default:
		logger.Error("Unknown int value of BulkResponseStatus. Got %d", r)
	}

	return strVal
}

func (r *BulkResponseStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func (c *BulkResponseStatus) UnmarshalJSON(b []byte) error {
	value := strings.Trim(string(b), `"`)
	switch value {
	case "created":
		*c = StatusCreated
	case "failed", "error":
		*c = StatusFailed
	case "exists":
		*c = StatusExists
	case "deleted":
		*c = StatusDeleted
	case "updated":
		*c = StatusUpdated
	case "equal":
		*c = StatusEqual
	default:
		return fmt.Errorf("unknown string value for BulkResponseStatus received from server. Got %q", value)
	}

	return nil
}

// BulkResponseData is a single data unit that was
// updated, deleted or created during the bulk operation.
// It consists of status information and the object
// itself
type BulkResponseData[T any] struct {

	// Status message of the operation
	Status BulkResponseStatus `json:"status"`

	// HTTP like status code of the operation
	StatusCode int `json:"code"`

	// The object that was handled by the bulk request
	Data T `json:"data"`

	// Optional error message if the StatusCode >= 300
	Error ErrorResponse
}

// String returns a pretty string with all debug information
// contained
func (r BulkResponse[T]) String() string {

	// Buld string of every single response data
	responseData := ""

	for i, d := range r.ResponseData {
		if i != 0 {
			responseData += fmt.Sprintf(" -------------- %d --------------\n", i)
		}

		responseData += fmt.Sprintf(
			`   Status  -> %s
    Code    -> %d
`, d.Status, d.StatusCode)

		if !d.Error.IsZero() {
			responseData += "    Error:\n" + d.Error.PrintLog("       ")
		}
	}

	return fmt.Sprintf(
		` Message  -> %s
 Success  -> %d
 Errors   -> %d
 Exists   -> %d
 ---------------- 0 ----------------
 %s`, r.Message, r.Overview.Successful, r.Overview.Errors, r.Overview.Exists, responseData,
	)
}

// WasSuccessful returns whether all requested operations
// were executed successfully from the api.
// This is the case when "Errors" and "Exists" are null
func (r BulkResponse[T]) WasSuccessful() bool {
	return r.Overview.Errors == 0 && r.Overview.Exists == 0
}
