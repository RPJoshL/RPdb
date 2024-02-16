package models

import "fmt"

// ErrorResponse represents a custom error returned from the PHP server
// if the response was erroneous (status code 3xx or 4xx).
//
// This struct is also used for 5xx errors. Only the message will be set
// in such a case with a string like "Unknown error".
//
// If a go error occurred the field "Error" will be set and the message
// field contains an empty string. Use always the "Error" function to get the client
// message!
type ErrorResponse struct {

	// Unique ID of the error
	ID string `json:"id"`

	// The "humanized" error message in the client language
	Message string `json:"message"`

	// An optional technical description of the error
	DetailedErrorDescription string `json:"detailedErrorDescription"`

	// The response code of the request
	ResponseCode int

	// The URL path relative to the base URL with the HTTP method
	Path string

	// Occurred go error (optional). If this field is given the Message is empty
	ErrorGo error

	// Only in server debug mode
	ErrorResponseDebug
}

// ErrorResponseDebug contains additional fields of the "ErrorResponse"
// which are only set when the debug mode was enabled in the server
// configuration
type ErrorResponseDebug struct {
	// Debug: Source line the error occured
	Line int `json:"line"`
	// Debug: Source file the error occured
	File string `json:"file"`

	// Debug: stack trace of the thrown exception
	Backtrace []string `json:"backtrace"`
}

func (err *ErrorResponse) Error() string {
	if err.Message == "" && err.ErrorGo != nil {
		return err.ErrorGo.Error()
	} else {
		return err.Message
	}
}

// PrintLog returns a string with all debug information
// contained. The errors are indented by the given string.
// The output looks like this:
//
// Message  -> MyMessage...
// ID       -> UNIQUE_ID
func (err *ErrorResponse) PrintLog(indent string) string {
	if err.Message == "" && err.ErrorGo != nil {
		return err.ErrorGo.Error()
	}

	statusHeader := ""
	if err.Path != "" {
		statusHeader += fmt.Sprintf("%sRequest for %s failed:\n", indent, err.Path)
	}
	rtc := fmt.Sprintf(
		`%s%sMessage -> %s
%sCode    -> %d
%sID      -> %s
`, statusHeader, indent, err.Message, indent, err.ResponseCode, indent, err.ID)

	// Collect optional debug information
	debug := ""
	if err.File != "" {
		debug += fmt.Sprintf("%sFile    -> %s#%d\n%sTrace   ->\n", indent, err.File, err.Line, indent)
		for i, c := range err.Backtrace {
			// 7 lines of debug backtracke are enough
			if i == 6 {
				break
			}

			debug += indent + "    " + c + "\n"
		}
	}

	return rtc + debug
}

// IsZero returns if the error response if empty
func (err *ErrorResponse) IsZero() bool {
	return err.ID == ""
}
