package models

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"git.rpjosh.de/RPJosh/go-logger"
)

// Use this value while filtering if you don't want to filter after a null string
// but either ignore the parameter while searching
const ParameterAnyValue = "<#~NotNULL~Any~#>"

// Entry represents a single executable data unit
type Entry struct {

	// Unique ID of the entry
	ID int `json:"id"`

	// Attribute of the entry
	Attribute *Attribute `json:"attribute"`

	// The date and time which was given by the entry creation
	DateTime DateTime `json:"date_time"`

	// At what time the entry should be executed. This time is calculated
	// by "DateTime + executionOffset" specified in the currently used token
	DateTimeExecution DateTime `json:"date_time_execution"`

	// An array with all parameters for the entry
	Parameters []EntryParameter `json:"parameters"`

	// The ID of the token which created the entry
	Creator int `json:"creator"`

	// Creation or updating only attributes //

	Message ResponseMessage `json:"message"`

	// Creation only: Instead of specifying an absolute time you can pass an offset to the current time.
	// This supports positive (+20) and negative (/20) values and should be followed by
	// a time unit like "s", "m", "h" and "d". This does also support the string "now".
	// E.g.: "+20m"
	Offset string `json:"offset" cli:"--offset,-off"`

	// Creation only: Set the seconds to zero in the time when using an "Offset"
	FullMinutes bool `json:"full_minutes" cli:"--fullMinutes,-fl"`

	// Creation only: the original provided date will be keept present when the offset of
	// the time "flows over" the day
	// E.g.:
	//  - Current time  = 2022-01-01T22:20:00
	//  - OffsetPattern = 2022-01-02T+5h:+0:+0
	//  - Created time  = 2022-01-02T03:20:00
	KeepDate bool `json:"keep_date_on_overflow" cli:"--keepDate,-kp"`

	// Creation only: Instead of specifying an absolute time you can pass an offset to the current time.
	// This is an extension to the field "Offset". It allows you to specify such an offset in all fields
	// of the ISO-8601 (without timezone) string.
	// You can also specify negative values (/20) and a weekday as the Date.
	// E.g.: "+0-05-06T+10:00:00", "+0-+1-+0T/5:00:+20",
	//        "Mo+2T20:00:00"        (week after the next)
	//        "2021-Mo2T20:00:00"    (calendar weeks of the year)
	//        "2021-01-Mo2T20:00:00" (week on a montly basis)
	OffsetPattern string `json:"offset_pattern" cli:"--datePattern,-dp"`

	// Exec Response //

	// Creation only (exec response): The maximum time to wait in seconds for a response (max: 60 seconds)
	Timeout NullInt `json:"timeout" cli:"--timeout,-t"`

	// Only for Exec Response: unique ID of the entry
	ExecutionResponseId int `json:"entry_id"`

	// Only for Exec Response: response code of the execution
	ResponseCode int `json:"response_code"`

	// Only for Exec Response: response message of the execution
	Response string `json:"response"`

	// For update: if the entry was already executed from this client.
	// This field may be nil. When the entry was received from the API, this field
	// is always present.
	// We use that structure to prevent warnings with nocopy
	execution *struct {
		WasExecuted atomic.Bool
	} `json:"-"`
}

// EntryParameter contains the value of a specific command line argument when looking at it
// from an execution view.
type EntryParameter struct {

	// Reference to the parameter of an entry.
	// For creation or update this field is not required. In that case the parameter
	// is obtained by the position of this EntryParameter within the 'parameters' array
	ParameterID int `json:"parameter_id"`

	// Raw value of the parameter or the name of a predefined parameter preset.
	// When a preset is used this field is null
	Value string `json:"value"`

	// Name of the parameter preset to use. This does override the 'value' property if set.
	// You can use this field to make sure that a preset is correctly used for creation / upate
	Preset string `json:"preset"`
}

// NewEntry decodes the JSON response of the given reader
// to a new Entry
func NewEntry(r io.Reader) *Entry {
	var ent Entry

	if err := json.NewDecoder(r).Decode(&ent); err != nil {
		logger.Warning("Failed to decode entry: %s", err)
	}

	// Initialize pointer value
	if ent.execution == nil {
		ent.execution = &struct{ WasExecuted atomic.Bool }{}
	}

	return &ent
}

// ToJson marshals this entry to a json string represented in bytes
func (e *Entry) ToJson() []byte {
	rtc, err := json.Marshal(e)
	if err != nil {
		logger.Warning("Failed to marshal entry filter: %s", err)
		return []byte("{}")
	} else {
		return rtc
	}
}

// UnmarshalJSON implements the unmarshal interface to set default values
// after unmarshal
func (e *Entry) UnmarshalJSON(data []byte) error {

	// We cannot use entry directly to avaoid a loop :)
	type TempEntry Entry

	var ee TempEntry
	if err := json.Unmarshal(data, &ee); err != nil {
		return err
	}
	*e = Entry(ee)

	// Initialize pointer value
	e.execution = &struct{ WasExecuted atomic.Bool }{}

	return nil
}

// DontIncludeParametersInRequest "omits" the field "Parameters" for patch API requests.
// This is a hack to keep the old parameters when no new parameters should be applied.
//
// This function will add an element to the Parameters array with special values that the
// API server will understand.
// It's not good but the only available option when we don't use a pointer / [sql.NullArray]
func (p *Entry) DontIncludeParametersInRequest() {
	p.Parameters = []EntryParameter{{Value: ParameterAnyValue + ParameterAnyValue}}
}

func (e Entry) String() string {
	parameter := ""
	if len(e.Parameters) == 1 {
		parameter += e.Parameters[0].GetDisplay(e.Attribute, false)
	} else if len(e.Parameters) != 0 {
		// Loop through all parameters and get display value
		for i, p := range e.Parameters {
			// Find parameter
			if e.Attribute == nil || i >= len(e.Attribute.Parameter) {
				parameter += fmt.Sprintf("\n    %-20s: %s", "<unknown>", p.GetDisplay(e.Attribute, false))
			}
			parameter += fmt.Sprintf("\n    %-20s: %s", e.Attribute.Parameter[i].Name, p.GetDisplay(e.Attribute, false))
		}
	}

	return fmt.Sprintf(
		`_____ %s (%d) _____
Parameter:  %s
Attribute:  %s
Execution:  %s
`, e.DateTime.FormatPretty(), e.ID, parameter, e.Attribute.Name, e.DateTimeExecution.FormatPretty(),
	)
}

func (e Entry) ToSlice() []string {
	rtc := []string{
		fmt.Sprintf("%d", e.ID),
		e.DateTime.Format(TimeFormat),
		e.Attribute.Name,
		e.DateTimeExecution.Format(TimeFormat),
	}

	// Add all parameter values
	for _, p := range e.Parameters {
		rtc = append(rtc, p.GetValue(e.Attribute))
	}

	return rtc
}

// GetParameterValue returns the value of this parameter that should be
// used for executing a script.
// This returns either the predefined parameter value or the raw value
func (ep *EntryParameter) GetValue(attribute *Attribute) string {
	if attribute != nil && ep.Preset != "" {
		for _, p := range attribute.Parameter {
			// Find parameter by ID
			if p.ID == ep.ParameterID {
				// Find preset for this parameter
				for _, pp := range p.Presets {
					if strings.EqualFold(pp.Name, ep.Preset) {
						return pp.Value
					}
				}
				logger.Warning("No parameter preset found within the attribute %q: %q", attribute.Name, ep.Preset)
			}
		}

		logger.Warning("No parameter with id %d found within the attribute %q: %q", ep.ParameterID, attribute.Name, ep.Preset)
		return ""
	} else {
		return ep.GetParameter()
	}
}

// GetParameter returns the raw parameter value of the field "Parameter".
// Null values are returned as an empty string
func (ep *EntryParameter) GetParameter() string {
	return ep.Value
}

// GetParameterDisplay returns the parameter value to show for the user.
// This is either the Field "Parameter" or the name / short name of the
// parameter preset
func (ep *EntryParameter) GetDisplay(attribute *Attribute, short bool) string {
	if attribute == nil || ep.Preset == "" {
		return ep.GetParameter()
	} else if !short {
		return ep.Preset
	} else {
		for _, p := range attribute.Parameter {
			// Find parameter by ID
			if p.ID == ep.ParameterID {
				// Find preset for this parameter
				for _, pp := range p.Presets {
					if strings.EqualFold(pp.Name, ep.Preset) {
						if pp.ShortName == "" {
							return pp.Name
						} else {
							return pp.ShortName
						}
					}
				}
				logger.Warning("No parameter preset found within the attribute %q: %q", attribute.Name, ep.Preset)
			}
		}

		logger.Warning("No parameter with id %d found within the attribute %q: %q", ep.ParameterID, attribute.Name, ep.Preset)
		return ep.Preset
	}
}

// IsPast returns weather all time fields of this entry lie in
// the past.
// You can specify if the execution time should be ignored
func (e *Entry) IsPast(ignoreExecutionTime bool) bool {

	// DateTime of the entry is not available. In such a case
	// "false" is always returned
	if e.DateTime.IsZero() {
		return false
	}

	return e.DateTime.Before(time.Now()) &&
		(ignoreExecutionTime || e.DateTimeExecution.IsZero() || e.DateTimeExecution.Before(time.Now()))
}

// WasExecuted states whether this entry was already executed
// in an execution context.
// Only use this function if the entry was created from the API and was not cloned!
func (e *Entry) WasExecuted() bool {
	if e.execution != nil {
		return e.execution.WasExecuted.Load()
	}

	logger.Debug("Called 'WasExecuted()' in entry #%d that was not initialized by the API", e.ID)
	return false
}

// SetExecuted stores the execution flag of the entry.
// Only use this function if the entry was created from the API and was not cloned!
func (e *Entry) SetExecuted(executed bool) {
	if e.execution != nil {
		e.execution.WasExecuted.Store(executed)
	} else {
		logger.Warning("Tried to store executed flag in an entry (#%d) that was not initialized by the API", e.ID)
	}
}

// ShouldExecuteNow returns if the entry should be executed now.
// This is the case if:
//   - the execution time of the entry is past and the attribute is of type exec_always
//   - the execution time (dateTime - now()) is within an offset of +0.5 seconds / -2.0 seconds
func (e *Entry) ShouldExecuteNow() bool {

	// If the entry got already executed return false
	if e.WasExecuted() {
		return false
	}

	// DateTime to use
	dateTime := e.DateTime
	if !e.DateTimeExecution.IsZero() {
		dateTime = e.DateTimeExecution
	}

	// No DateTime provided so return true
	if dateTime.IsZero() {
		return false
	}

	// Always return true for old entries of the type "exec_always"
	if e.Attribute.ExecuteAlways && dateTime.Before(time.Now()) {
		return true
	}

	// Is the offset to the current time within a range of 1.5 seconds?
	offset := time.Until(dateTime.Time).Seconds()
	return offset <= 0.5 && offset >= -2
}

// GetExecutionTime returns the DateTime to which the entry should be
// executed.
// This is either the field DateTimeExecution or if that is zero
// (or you have passed the ignoring of the execution time) the field dateTime
func (e *Entry) GetExecutionTime(ignoreExecutionTime bool) time.Time {
	if ignoreExecutionTime || e.DateTimeExecution.Time.IsZero() {
		return e.DateTime.Time
	} else {
		return e.DateTimeExecution.Time
	}
}

// SetFullMinutes sets the flag "fullMinutes" to 'true'
func (e *Entry) SetFullMinutes() string {
	e.FullMinutes = true
	return ""
}

// SetKeepDate sets the flag "keepDate" to 'true'
func (e *Entry) SetKeepDate() string {
	e.KeepDate = true
	return ""
}

// SetTimeout sets the field "timeout" to the given value
func (e *Entry) SetTimeout(val string) string {
	if n, err := strconv.Atoi(val); err != nil {
		return fmt.Sprintf("Failed to conver the value %q to a number", val)
	} else {
		e.Timeout = NullInt{Valid: true, Int32: int32(n)}
	}

	return ""
}

// ExecutionResponse returns a nicely formatted string of the
// execution response if the attribute of the entry was of the type
// "exec response"
func (e *Entry) ExecutionResponse() string {
	return fmt.Sprintf(`
----------------------------------------
Response code: %d

%s`, e.ResponseCode, e.Response)
}

// NewParameter is a helper function to create a single Parameter
// easilier on the fly with a single method call
func NewParameters(value, preset string) []EntryParameter {
	return []EntryParameter{{Value: value, Preset: preset}}
}
