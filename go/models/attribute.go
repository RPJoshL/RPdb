package models

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"git.rpjosh.de/RPJosh/go-logger"
)

const (
	PARAMETER_TYPE_STRING = "text"
	PARAMETER_TYPE_NUMBER = "number"
	PARAMETER_TYPE_BOOL   = "boolean"
)

// Attribute is used for grouping entries to a shared "executable"
// operation.
// For each entry an attribute is required
type Attribute struct {

	// Unique ID of the attribute
	ID int `json:"id"`

	// Name of the attribute. This is unique within the user account
	Name string `json:"name"`

	// If EA is enabled for the attribute, an entry belonging to this attribute
	// is always executed. Even if the date is past the entry will be returned by
	// the server until you register the entry as executed
	ExecuteAlways bool `json:"execute_always"`

	// The entry will not be saved in the database. It is only sent once
	// over the websocket connection
	NoDb bool `json:"no_db"`

	// A response message and code is expected to be returned to the client
	// immediately after the entry was executed
	ExecResponse AttributeExecResponse `json:"execution_response"`

	// Rights of the currently authenticated token for entries created with this attribute
	Rights Right `json:"rights"`

	// Default right to apply if the right is not overwritten by a specific token configuration.
	// It does overwrite the global token rights only if it is set to `all` or the attribute ones are set to `none`
	DefaultRight Right `json:"default_right"`

	// A list of parameters taht are available for this attribute
	Parameter []AttributeParameter `json:"parameters"`

	// Field that is used from the API to sort the attributes ascendant after this value
	SortOrder int `json:"sort_order"`
}

// AttributeParameter specifies the number and order of parameters that can
// be used while creating an entry.
// In an execution context, these are the arguments that are used while calling the program.
// You can create up to six possible parameters for an attribute
type AttributeParameter struct {

	// Unique ID of the parameter
	ID int `json:"id"`

	// Unique name of the parameter within the attribute
	Name string `json:"name"`

	// Position of the parameter in an execution context.
	// This is also the order in which the parameters are passed
	// in an entry.
	// Possible values: 1 - 6
	Position int `json:"position"`

	// Data type of the parameter. See constants 'PARAMETER_TYPE' for possible values
	Type string `json:"type"`

	// Force the usage of a predefinded value for this parameter
	ForcePreset bool `json:"force_preset"`

	// Predefined values for this parameter
	Presets []ParameterPreset `json:"presets"`
}

// ParameterPreset is a object that countains predefined values
// for an pamarter of an attribute.
// These can be used during the creation of an entry to make the management of
// available values for an parmaeter easier
type ParameterPreset struct {

	// Unique name of the preset within the parameter
	Name string `json:"name"`

	// A short "abbrevation" name of the unique preset name
	ShortName string `json:"name_short"`

	// The underlaying value of the parameter that should be used for the executions
	Value string `json:"value"`

	// Field that is used from the API to sort the parameters ascendant after this value
	SortOrder int `json:"sort_order"`
}

// AttributeExecResponse contains all information for the attribute type
// "exec_response" for which a response message and a code is expected to be
// returned to the client
type AttributeExecResponse struct {

	// Weather this function is enabled
	Enabled bool `json:"enabled"`

	// When this toggle is set an entry for this attribute can also be scheduled delayed.
	// By default the execution time has to be "now"
	AllowDelayedExecution bool `json:"allow_delayed_execution"`

	// The default time to wait for an execution response
	DefaultTimeout int `json:"default_timeout"`
}

// Rights of the currently authenticated token for entries created with this attribute
type Right int

const (
	NONE Right = -1
	ALL        = iota
	READ
	WRITE
)

func (c *Right) UnmarshalJSON(b []byte) error {
	value := strings.Trim(string(b), `"`)
	if value == "" || value == "null" {
		return nil
	}

	switch value {
	case "none":
		*c = NONE
	case "all":
		*c = ALL
	case "read":
		*c = READ
	case "write":
		*c = WRITE
	default:
		return fmt.Errorf("invalid right received: %s", value)
	}

	return nil
}

func (c Right) String() string {
	switch c {
	case NONE:
		return "none"
	case ALL:
		return "all"
	case READ:
		return "read"
	case WRITE:
		return "write"
	case 0:
		// Return "all" by default
		return "all"
	default:
		logger.Warning("Tried to parse invalid right value: %d", c)
		return "all"
	}
}

func (c Right) MarshalJSON() ([]byte, error) {
	return []byte(`"` + c.String() + `"`), nil
}

// NewAttribute decodes the JSON response of the given reader
// to a new attribute
func NewAttribute(r io.Reader) *Attribute {
	var attr Attribute

	if err := json.NewDecoder(r).Decode(&attr); err != nil {
		logger.Warning("Failed to decode attribute: %s", err)
	}

	return &attr
}

func (ap AttributeParameter) String(indent string) string {
	// Build info string for presets
	presets := ""
	if len(ap.Presets) > 0 {
		presets = ":"
		for _, pres := range ap.Presets {
			presets += "\n" + pres.String(indent+"    -> ")
		}
	}

	// Properties of the paramter
	props := ap.Type
	if ap.ForcePreset {
		props += ",preset"
	}

	return fmt.Sprintf("%s#%d %s (%s)%s\n", indent, ap.Position, ap.Name, props, presets)
}

func (pp ParameterPreset) String(indent string) string {
	name := pp.Name

	if pp.ShortName != "" {
		name += fmt.Sprintf(" (%s)", pp.ShortName)
	}

	return fmt.Sprintf("%s%-15s %s", indent, name+":", pp.Value)
}

func (a Attribute) String() string {
	// Build parameter string
	parameter := ""
	for _, p := range a.Parameter {
		parameter += p.String("    ")
	}

	// Build (extended) exec_response string
	execResponse := ""
	if a.ExecResponse.Enabled {
		execResponse += fmt.Sprintf(" (with timeout of %d seconds", a.ExecResponse.DefaultTimeout)
		if a.ExecResponse.AllowDelayedExecution {
			execResponse += " and delayed execution"
		}
		execResponse += ")"
	}

	return fmt.Sprintf(
		`_______ %s (%d) _______
Execute always:  %t
No DB:           %t
Exec Response:   %t%s
Rights:          %s
Parameter:
%s
`, a.Name, a.ID, a.ExecuteAlways, a.NoDb, a.ExecResponse.Enabled, execResponse, a.Rights, parameter)
}

func (a Attribute) ToSlice() []string {
	return []string{
		fmt.Sprintf("%d", a.ID),
		a.Name,
		strconv.FormatBool(a.ExecuteAlways),
		strconv.FormatBool(a.NoDb),
		strconv.FormatBool(a.ExecResponse.Enabled),
	}
}
