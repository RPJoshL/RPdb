package models

import (
	"encoding/json"
	"strings"
	"time"

	"git.rpjosh.de/RPJosh/go-logger"
)

// EntryFilter is used to filter entries based on their
// properties.
// In addition to the fields of an entry some extra filters
// can be used to make the life of you easier
type EntryFilter struct {
	// Only entries with an ID that is contained inside the array are fetched
	IDs []int `json:"ids" cli:"--ids,-i"`

	// Only entries that are having an attribute included in the list
	Attributes []int `json:"attribute"`

	// Filter for the value or after the name of a parameter preset.
	// The parameter ID is determined by its position in the provided Array.
	// A null value for a member in this array means that the parameter at the arrays
	// position can have any value. An empty string ("") means that you want to filter after
	// null values.
	// If this field is nil, the parametes are not filtered. When this array is empty,
	// only entries which do not have any parameter / all parameters are null will be returned
	Parameters *[]NullString `json:"parameters"`

	// Only entries that were created by the given ID of the API key are returned
	Creator int `json:"creator"`

	// An offset like in "Entry" supporting wildcards in it
	DatePattern string `json:"pattern" cli:"--datePattern,-dp"`

	// The date of the entry has to be later than the given date.
	// This can either be a time.Time formatted in the server time format
	// or an offset
	LaterThan string `json:"later_than" cli:"--laterThan,-lt"`
	// The date of the entry has to be earlier than the given date.
	// This can either be a time.Time formatted in the server time format
	// or an offset
	EarlierThan string `json:"earlier_than" cli:"--earlierThan,-et"`

	// Return also entries that are already past (dateTime < now).
	// This value does not override the "executeAlways" behaviour
	OldDates bool `json:"old_dates" cli:"--oldDates,-od,~~~"`

	// Ignore the flag "execute_always" for all attributes
	IgnoreEA bool `json:"ignore_execute_always"`

	// Ignore the flag "execute_always" for all attributes that are contained
	// in this list
	IgnoreEAAttribute []int `json:"ignore_execute_always_attribute"`

	// The maximum amount of entries to return. Maximum value are 200
	MaxEntries int `json:"max_entries" cli:"--max,-m"`

	// A list of entries with the flag "execute_always" that has already been
	// executed from this client
	Executed []int `json:"executed"`

	// Controls which date field should be used to filter the date:
	//  0 = "date_time > now() || date_time_execution > now()"
	//  1 = "date_time > now()"
	//  2 = "date_time_execution > now()"
	IgnoreExecutionDate int
}

func (e *EntryFilter) ToJson() []byte {

	// Replace a NULL parameter with the search string for any parameter
	if e.Parameters != nil {
		for i, p := range *e.Parameters {
			// Null values are replaced by any value
			if !p.Valid {
				(*e.Parameters)[i] = NewNullString(ParameterAnyValue)
			}
		}
	}

	rtc, err := json.Marshal(e)
	if err != nil {
		logger.Warning("Failed to marshal entry filter: %s", err)
		return []byte("{}")
	} else {
		return rtc
	}
}

// CanHandleLocally returns whether the filtering can be handled
// locally without calling the API by simple "==" comparisons
func (e *EntryFilter) CanHandleLocally() bool {
	return true &&
		e.DatePattern == "" &&
		e.LaterThan == "" &&
		e.EarlierThan == "" &&
		!e.OldDates // No old dates are fetched by default
}

// IsZero checks if this filter is empty and contains
// no filter condition
func (e *EntryFilter) IsZero() bool {
	return true &&
		len(e.IDs) == 0 &&
		len(e.Attributes) == 0 &&
		e.Parameters == nil &&
		e.Creator == 0 &&
		e.DatePattern == "" &&
		e.LaterThan == "" &&
		e.EarlierThan == "" &&
		!e.OldDates &&
		!e.IgnoreEA &&
		len(e.IgnoreEAAttribute) == 0 &&
		e.MaxEntries == 0 &&
		len(e.Executed) == 0 &&
		e.IgnoreExecutionDate == 0
}

// DoesMatch checks if the filter matches for the given entry.
// Note that a correct result is only returned if all fields
// can be handled locally.
// Use the function "CanHandleLocally()" to check that
func (e *EntryFilter) DoesMatch(ent Entry) bool {

	// Validate that the entry is contained in the provided filter list
	if len(e.IDs) != 0 {
		wasFound := false
		for _, id := range e.IDs {
			if id == ent.ID {
				wasFound = true
				break
			}
		}
		if !wasFound {
			return false
		}
	}

	// Check if the attribute does match
	if len(e.Attributes) != 0 {
		wasFound := false
		for _, id := range e.Attributes {
			if id == ent.Attribute.ID {
				wasFound = true
				break
			}
		}
		if !wasFound {
			return false
		}
	}

	// Validate parameter
	if e.Parameters != nil {
		for i, p := range ent.Parameters {
			// No parameter to compare against anymore → the parameters are equal
			if i >= len(*e.Parameters) {
				break
			}
			filterP := (*e.Parameters)[i]

			// Accept any value
			if !filterP.Valid || filterP.String == ParameterAnyValue {
				continue
			}

			// Compare parameter value / preset
			if (p.Value != "" && p.Value == filterP.String) || (p.Preset != "" && strings.EqualFold(p.Preset, filterP.String)) {
				// Parameters are equal
			} else if filterP.String == "" && p.Preset == "" && p.Value == "" {
				// Both the filter parameter and entry parameter are null
			} else if ent.Attribute != nil && p.Preset != "" && len(ent.Attribute.Parameter) > i {
				// Check if the value equals the value of the parameter preset of the entry
				for _, app := range ent.Attribute.Parameter[i].Presets {
					// Find preset with the name
					if strings.EqualFold(app.Name, p.Preset) {
						return app.Value == filterP.String
					}
				}

				return false
			} else {
				return false
			}
		}
	}

	// Validate creator
	if e.Creator != 0 {
		if e.Creator != ent.Creator {
			return false
		}
	}

	// Ignore entries with the flag EA that are laying in the past
	shouldIgnoreEA := e.IgnoreEA
	for _, e := range e.IgnoreEAAttribute {
		if e == ent.Attribute.ID {
			shouldIgnoreEA = true
		}
	}

	// Check the execution date
	if e.IgnoreExecutionDate == 0 {
		if ent.DateTime.Time.Before(time.Now()) && ent.DateTimeExecution.Time.Before(time.Now()) && (!ent.Attribute.ExecuteAlways || shouldIgnoreEA) {
			return false
		}
	} else if e.IgnoreExecutionDate == 1 {
		if ent.DateTime.Time.Before(time.Now()) && (!ent.Attribute.ExecuteAlways || shouldIgnoreEA) {
			return false
		}
	} else if e.IgnoreExecutionDate == 2 {
		if ent.DateTimeExecution.Time.Before(time.Now()) && (!ent.Attribute.ExecuteAlways || shouldIgnoreEA) {
			return false
		}
	}

	return true
}

// SetOldDates sets the flag "oldDates" to 'true'
func (e *EntryFilter) SetOldDates() string {
	e.OldDates = true
	return ""
}
