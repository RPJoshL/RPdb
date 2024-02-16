package args

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	mod "github.com/RPJoshL/RPdb/v4/go/models"
	"git.rpjosh.de/RPJosh/go-logger"
)

func (e *EntryList) Help() string {
	return `
list [options]		|Lists all available entries matching the filter options
    
    --ids         -i {id,id}     |Entries with the given ids
    --attribute   -a {id\|name}  |Entries with the given attributes (seperated by ',')
    --parameter   -p [ 1 2 ]     |Entries with the given parameter value or the name of a preset will be returend.
                                 The order of the parameters matches the position within the given array.
                                 Hint: use '<#~NotNULL~Any~#>' to search for any value
    --datePattern -dp {pat}      |The date will be filtered by the given pattern (with wildcard 
                                 support for '.' and '*'). Examples:
                                 |Next week Monday to the same (full) hour: Mo+1T+0:00:00
                                 Yesterday at 05:00: +0-+0-/1T05:00:00  ~  Past two hours: /2h
    --oldDates     -od           |Old dates will be returned. This is by default only the case
                                 for attributes with 'execute always' that are not yet executed
    --earlierThan  -et {xx}      |The date has to be earlier than the given value. Pattern is possible
    --laterThan    -lt {xx}      |The date has to be earlier than the given value. Pattern is possible

    --max          -m  {x}       |Shows at a max rate {x} entries
    --count        -c            |Shows only the NUMBER of entries (-1 on error)
|_______________________________________________________________________________

Global options that can be used for almost all comamnds.
	
    --output  {format}        |Output format to use|. Available formats are 'pretty', 'json' and 'csv'
`
}

func (e *EntryCreate) Help() string {
	return `
create -a\|--attribute id\|name {one of the available method} [options]

    --attribute -a  {id\|name} |Attribute for the entry
    --date      -d  {date}    |Date in the ISO format (YYY-MMM-DDThh:mm:ss)
    --offset    -off {offset} |Positive time offset to the current date and time.
                              |Allowed units are 's', 'm', 'h' and 'd': +20m = in 20 minutes
        --fullMinutes -fl     |The seconds will be set to '00'
    --datePattern -dp {pat}   |Extended method of the offset| for which every field (year, month, ...)
                              an own offset (positive: + \| negative: /) can be given.
                              Examples: 2021-01-01T+20:00:00  \|  +0-+1-+0T/5:00:+20
        --keepDate    -kd     |The date will be kept during overflow of the date (day will not be changed)

    --parameter -p  [ 1 2 ]   |Parameter values or the name of a preset for the entry
    --timeout   -t  {sec}     |Exec Response: Waiting time in seconds to receive a response.
                              |Specify "0" to not wait for an answer
|_______________________________________________________________________________

Global options that can be used for almost all comamnds.

    --output  {format}        |Output format to use|. Available formats are 'pretty', 'json' and 'csv'
`
}

func (e *EntryDelete) Help() string {
	return fmt.Sprintf(
		`
delete [options]    |Delete entries base on the given search parameters
                    |See the section "list" for options
%s`, regexp.MustCompile(`^.*\n.*\n`).ReplaceAllString((&EntryList{}).Help(), ""))
}

func (e *EntryUpdate) Help() string {
	return fmt.Sprintf(
		`
update id,id,id  {fields}   |For all the given entries the fields will be updated accordingly
%s`, regexp.MustCompile(`^.*\n.*\n`).ReplaceAllString((&EntryCreate{}).Help(), ""))
}

func (e *Entry) Help() string {
	return (`
Create, delete, update and query entries.

list [options]		        |Lists all available entries matching the filter options

delete [options]            |Delete entries base on the given search parameters
                            |See the section "list" for options

create -a\|--attribute id\|name {one of the available method} [options] | Create a single entry

update id,id,id  {fields}   |For all the given entries the fields will be updated accordingly
|_______________________________________________________________________________

|Global options that can be used for almost all comamnds.
	
    --output  {format}  	Output format to use. Available formats are 'pretty', 'json' and 'csv'
	`)
}

func (e *Entry) GetAttributeNames(cli *Cli, input string) (rtc []string) {
	rtc = make([]string, 0)

	if attributes, err := cli.GetApi().GetAttributes(); err != nil {
		logger.Error("[Autocomplte] Failed to fetch attributes: %s", err)
	} else {
		for _, a := range attributes {
			rtc = append(rtc, a.Name)
		}
	}

	return rtc
}

func (e *EntryCreate) GetAttributeNames(cli *Cli, input string) (rtc []string) {
	return cli.Entry.GetAttributeNames(cli, input)
}

func (e *EntryList) GetAttributeNames(cli *Cli, input string) (rtc []string) {
	return cli.Entry.GetAttributeNames(cli, input)
}

func (e *EntryList) GetParameters(cli *Cli, input []string) (rtc []string) {

	// Maximum number of parameters are six
	if len(input) > 6 {
		return []string{"]"}
	}

	// Parameter preset or the data type can only be autocompleted when attribute was given
	if e.Attributes == "" {
		return []string{}
	}

	// Get the current input to autocomplate
	in := ""
	pos := len(input) - 1
	if len(input) != 0 {
		in = input[pos]
	}

	// Get Parameter presets
	rtc = append(rtc, e.GetParameterPresets(cli, in, pos)...)

	return rtc
}

func (e *EntryCreate) GetParameters(cli *Cli, input []string) (rtc []string) {
	// Maximum number of parameters are six
	if len(input) > 6 {
		return []string{"]"}
	}

	// Parameter preset or the data type can only be autocompleted when attribute was given
	if e.Attribute == "" {
		return []string{}
	}

	// Get the current input to autocomplate
	in := ""
	pos := len(input) - 1
	if len(input) != 0 {
		in = input[pos]
	}

	// Get Parameter presets
	rtc, attr := GetParameterPresets(cli, in, e.Attribute, pos)

	// Maximum number of parameters exceeded
	if attr != nil && pos >= len(attr.Parameter) {
		return []string{"]"}
	}

	return rtc
}

func (e *EntryCreate) CanOptionBeUsedForComplete(longKey string) bool {
	// Only one date function can be used
	if longKey == "--date" || longKey == "--datePattern" || longKey == "--offset" {
		return e.Entry.DateTime.IsZero() && e.Entry.Offset == "" && e.Entry.OffsetPattern == ""
	}

	// Full minutes can only be used for offset and date pattern
	if longKey == "--fullMinutes" || longKey == "--keepDate" {
		return e.Entry.Offset != "" || e.Entry.OffsetPattern != ""
	}

	return true
}

func (e *EntryCreate) GetParameterPresets(cli *Cli, input string) (rtc []string) {
	rtc, _ = GetParameterPresets(cli, input, e.Attribute, -1)
	return
}
func (e *EntryList) GetParameterPresets(cli *Cli, input string, position int) (rtc []string) {
	rtc = make([]string, 0)

	// Loop through every parameter and get autocomplete values
	i := 0
	maxCountParameters := 0
	for _, attr := range strings.Split(e.Attributes, ",") {
		presets, a := GetParameterPresets(cli, input, attr, position)
		rtc = append(rtc, presets...)

		// Increment counter and cache the maximum number of parameters for all attributes
		i++
		if a != nil && len(a.Parameter) > maxCountParameters {
			maxCountParameters = len(a.Parameter)
		}
	}

	// Maximum number of parameters exceeded for all attributes
	if position >= maxCountParameters {
		return []string{"]"}
	}

	return rtc
}

// GetParameterPresets returns all available parameter presets for the attribute and the parameter.
// The parameters position is indexed by 0
func GetParameterPresets(cli *Cli, input string, attribute string, position int) (rtc []string, attr *mod.Attribute) {
	rtc = make([]string, 0)

	if attribute != "" {
		if id, err := strconv.Atoi(attribute); err == nil {
			if attr, err := cli.GetApi().GetAttribute(id); err != nil {
				logger.Error("[Autocomplete] Failed to fetch attribute %q: %s", attribute, err)
			} else {
				if position < len(attr.Parameter) {
					for _, par := range attr.Parameter[position].Presets {
						rtc = append(rtc, par.Name)
					}

					// Add true / false for boolean parameter
					if !attr.Parameter[position].ForcePreset && attr.Parameter[position].Type == mod.PARAMETER_TYPE_BOOL {
						rtc = append(rtc, "true", "false")
					}
				}

				return rtc, attr
			}
		} else {
			if attr, err := cli.GetApi().GetAttributeByName(attribute); err != nil {
				logger.Error("[Autocomplte] Failed to fetch attribute %q: %s", attribute, err)
			} else {
				if position < len(attr.Parameter) {
					for _, par := range attr.Parameter[position].Presets {
						rtc = append(rtc, par.Name)
					}

					// Add true / false for boolean parameter
					if !attr.Parameter[position].ForcePreset && attr.Parameter[position].Type == mod.PARAMETER_TYPE_BOOL {
						rtc = append(rtc, "true", "false")
					}
				}

				return rtc, attr
			}
		}
	}

	return rtc, nil
}

func (e *EntryCreate) GetOutputFormats(cli *Cli, input string) (rtc []string) {
	return []string{"pretty", "csv", "json"}
}
func (e *EntryList) GetOutputFormats(cli *Cli, input string) (rtc []string) {
	return []string{"pretty", "csv", "json"}
}
