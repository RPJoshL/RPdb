package models

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"git.rpjosh.de/RPJosh/go-logger"
)

// Date format the server understands and accepts
const TimeFormat = "2006-01-02T15:04:05"

// Pretty time format for printing and debugging
const TimeFormatPretty = "02.01.2006 15:04:05"

// DateTime is a time.Time object wrapped by custom marshal options
// to handle the parsing of the server time format
type DateTime struct {
	time.Time
}

func (c *DateTime) UnmarshalJSON(b []byte) error {
	value := strings.Trim(string(b), `"`)
	if value == "" || value == "null" {
		return nil
	}

	// The date is always returned to the client's timezone.
	// Therefore, set the time zone of this time to the client timezone
	t, err := time.ParseInLocation(TimeFormat, value, time.Now().Location())
	if err != nil {
		return err
	}

	*c = DateTime{t}
	return nil
}

func (c DateTime) MarshalJSON() ([]byte, error) {
	if c.IsZero() {
		return []byte("null"), nil
	} else {
		return []byte(`"` + time.Time(c.Time).Format(TimeFormat) + `"`), nil
	}
}

// NewDateTime returns a wrapped time.Time object
// based on the given time string.
// The format of the time has to be like this:
// "2022-08-22T14:00:12" (see TimeFormat)
func NewDateTime(dateTime string) DateTime {
	t, err := time.Parse(TimeFormat, dateTime)
	if err != nil {
		logger.Error("Unable to parse the given DateTime: %q", dateTime)
	}

	return DateTime{Time: t}
}

// ConvertDateTime converts the given time.Time struct into
// the custom wrapper DateTime
func ConvertDateTime(dateTime time.Time) DateTime {
	return DateTime{Time: dateTime}
}

func (c DateTime) FormatPretty() string {
	return c.Format(TimeFormatPretty)
}

// NullString is a wrapper around sql.NullString that is encoded to
// a null on json marshal if it is not valid
type NullString sql.NullString

func (x *NullString) MarshalJSON() ([]byte, error) {
	if !x.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(x.String)
}

func (c *NullString) UnmarshalJSON(b []byte) error {
	value := strings.Trim(string(b), `"`)
	if value == "" || value == "null" || value == ParameterAnyValue {
		c.Valid = false
	} else {
		c.Valid = true
		c.String = value
	}

	return nil
}

// NewNullString creates a new sql.NullString with the given
// parameter. If the parameter is empty, the string gets
// converted to NULL during any api interaction
func NewNullString(str string) NullString {
	return NullString{
		Valid:  str != "",
		String: str,
	}
}

// NullString is a wrapper around sql.NullInt32 that is encoded to
// a null on json marshal if it is not valid
type NullInt sql.NullInt32

func (x *NullInt) MarshalJSON() ([]byte, error) {
	if !x.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(x.Int32)
}

func (c *NullInt) UnmarshalJSON(b []byte) error {
	value := strings.Trim(string(b), `"`)
	if value == "" || value == "null" || value == "0" {
		c.Valid = false
	} else {
		c.Valid = true

		if n, err := strconv.Atoi(value); err != nil {
			return err
		} else {
			c.Int32 = int32(n)
		}
	}

	return nil
}

// NewNullInt creates a new sql.NullInt32 with the given
// parameter. If the parameter is "0", the string gets
// converted to NULL during any api interaction
func NewNullInt(n int) NullInt {
	return NullInt{
		Valid: n != 0,
		Int32: int32(n),
	}
}
