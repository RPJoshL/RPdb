package args

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	mod "github.com/RPJoshL/RPdb/v4/go/models"
)

// Entry contains entry options for the CLI
type Entry struct {
	Disabled    bool
	EntryList   EntryList   `cli:"list,l"`
	EntryDelete EntryDelete `cli:"delete,d"`
	EntryCreate EntryCreate `cli:"create,c"`
	EntryUpdate EntryUpdate `cli:"update,u"`
}

type EntryList struct {
	// Pass CLI parameters from entry filter directly
	EntryFilter mod.EntryFilter `cli:","`

	Attributes   string   `cli:"--attribute,-a" completion:"GetAttributeNames"`
	Parameter    []string `cli:"--parameter,-p" completion:"GetParameters"`
	ParameterSet bool

	Count bool `cli:"--count,-c,~~~"`

	Format string `cli:"--output,-o" completion:"GetOutputFormats"`
}

type EntryDelete struct {
	// Pass CLI parameters from EntryList directly
	EntryList EntryList `cli:","`
}

type EntryCreate struct {
	// Pass CLI parameters from entry
	Entry mod.Entry `cli:","`

	Attribute    string   `cli:"--attribute,-a" completion:"GetAttributeNames"`
	Date         string   `cli:"--date,-d"`
	Parameter    []string `cli:"--parameter,-p" completion:"GetParameters"`
	ParameterSet bool

	Format string `cli:"--output,-o" completion:"GetOutputFormats"`
}

type EntryUpdate struct {
	// Pass CLI parameters from EntryCreate directly
	EntryCreate EntryCreate `cli:","`

	// IDs to update
	IDs []int `cli:"--ids,-i,,1"`
}

func (e *EntryList) SetCount() string {
	e.Count = true

	return ""
}

func (e *EntryList) SetParameter(parameters []string) string {
	e.ParameterSet = true
	e.Parameter = parameters

	return ""
}

func (e *EntryCreate) SetParameter(parameters []string) string {
	e.ParameterSet = true
	e.Parameter = parameters

	return ""
}

// ApplyFillter applies the dynamic filter parameters from the cli to the
// EntryFilter
func (e *EntryList) ApplyFilter(cli *Cli) string {
	// Filter after attribute IDs
	if e.Attributes != "" {
		// Get all attributes to resolve attribute names
		attributes, err := cli.GetApi().GetAttributes()
		if err != nil {
			return cli.PrintFatalErrorf("Failed to fetch available attributes: %s", err)
		}

	outer:
		for _, val := range strings.Split(e.Attributes, ",") {
			id := -1
			if intVal, err := strconv.Atoi(val); err == nil {
				id = intVal
			}

			// Loop through all attributes to find id / name
			for _, a := range attributes {
				if a.ID == id || a.Name == val {
					e.EntryFilter.Attributes = append(e.EntryFilter.Attributes, a.ID)
					continue outer
				}
			}

			// No matching attribute found
			return cli.PrintFatalErrorf("No attribute found for id / name %q", val)
		}
	}

	// Set parameters (by position)
	if e.ParameterSet {
		paramaeters := make([]mod.NullString, len(e.Parameter))
		for i, p := range e.Parameter {
			paramaeters[i] = mod.NewNullString(p)
			// The parameter is valid because it was provided (not null)
			paramaeters[i].Valid = true
		}
		e.EntryFilter.Parameters = &paramaeters
	}

	return ""
}

func (e *EntryList) SetEntryList(cli *Cli) string {
	e.ApplyFilter(cli)

	// Make the request
	entries, err := cli.GetApi().GetEntries(e.EntryFilter)
	if err != nil {
		return cli.PrintFatalError(err.Error())
	}

	// Only print the number of entries
	if e.Count {
		fmt.Printf("%d\n", len(entries))
		return ""
	}

	// Print the entries (always as array)
	cli.PrintEntriesFormatted(entries, e.Format)
	return ""
}

func (e *EntryDelete) SetEntryDelete(cli *Cli) string {
	e.EntryList.ApplyFilter(cli)

	// Make the request
	deleted, err := cli.GetApi().DeleteEntriesFiltered(e.EntryList.EntryFilter)
	if err != nil {
		return cli.PrintFatalError(err.Error())
	}

	// Only print the number of deleted entries
	if e.EntryList.Count {
		fmt.Printf("%d\n", deleted.Count)
		return ""
	}

	// Print the result of the deletion
	switch strings.ToUpper(e.EntryList.Format) {
	case "PRETTY", "":
		fmt.Println(deleted.Message)
	case "CSV":
		w := csv.NewWriter(os.Stdout)
		w.Write([]string{fmt.Sprintf("%d", deleted.Count), deleted.Message.Client})
		w.Flush()
	case "JSON":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(deleted)
	default:
		cli.PrintFatalErrorf("Invalid format given: %q", e.EntryList.Format)
	}

	return ""
}

// PrintEntriesFormatted is a helper function to convert from []*mod.Entry to
// []mod.Formattable
func (cli *Cli) PrintEntriesFormatted(entries []*mod.Entry, format string) {
	rtc := make([]mod.Formattable, len(entries))

	for i, e := range entries {
		rtc[i] = e
	}

	cli.PrintStructsFormatted(&rtc, format)
}

func (e *EntryCreate) SetDate(val string) string {
	// Try to parse the time
	if tme, err := time.Parse(mod.TimeFormat, val); err != nil {
		return "Failed to parse the time '" + val + "'. Expected the format YYYY-MM-DDThh:mm:ss"
	} else {
		e.Entry.DateTime = mod.DateTime{Time: tme}
	}

	return ""
}

// ApplyEntry applies the dynamic entry parameters from the CLI to the
// Entry
func (e *EntryCreate) ApplyEntry(cli *Cli) string {

	// Build entry parameters from input. All parameters are passed by position
	for _, p := range e.Parameter {
		e.Entry.Parameters = append(e.Entry.Parameters, mod.EntryParameter{Value: p})
	}

	if e.Attribute != "" {
		var idInt = -1
		// Try to parse the attribute to an ID
		if intVal, err := strconv.Atoi(e.Attribute); err == nil {
			idInt = intVal
		}

		// Get all attributes for the user
		attributes, err := cli.GetApi().GetAttributes()
		if err != nil {
			return cli.PrintFatalErrorf("Failed to fetch available attributes: %s", err)
		}

		// Search for the attribute Name
		for _, a := range attributes {
			if a.ID == idInt || a.Name == e.Attribute {
				e.Entry.Attribute = a
				break
			}
		}

		if e.Entry.Attribute.ID == 0 {
			return cli.PrintFatalErrorf("Unable to find attribute with name %q", e.Attribute)
		}

	}

	return ""
}

func (e *EntryCreate) SetEntryCreate(cli *Cli) string {
	e.ApplyEntry(cli)

	// Attribute is required
	if e.Entry.Attribute == nil {
		return cli.PrintFatalError("Required parameter '--attribute' is missing")
	}

	ent, err := cli.GetApi().CreateEntry(e.Entry)
	if err != nil {
		return cli.PrintFatalError(err.Error())
	}

	if e.Entry.Attribute.ExecResponse.Enabled && (!e.Entry.Attribute.ExecResponse.AllowDelayedExecution || ent.ExecutionResponseId != 0) {
		// Return execution response
		switch strings.ToUpper(e.Format) {
		case "PRETTY", "":
			fmt.Println(ent.ExecutionResponse())
		case "CSV":
			w := csv.NewWriter(os.Stdout)
			w.Write([]string{fmt.Sprintf("%d", ent.ResponseCode), ent.Response})
			w.Flush()
		case "JSON":
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(struct {
				Code     int                 `json:"code"`
				Response string              `json:"response"`
				Message  mod.ResponseMessage `json:"message"`
			}{Code: ent.ResponseCode, Response: ent.Response, Message: ent.Message})
		default:
			return cli.PrintFatalErrorf("Invalid format given: %q", e.Format)
		}
	} else {
		switch strings.ToUpper(e.Format) {
		case "PRETTY", "":
			fmt.Println(ent.Message.Client)
		case "CSV", "JSON":
			cli.PrintStructFormatted(ent, e.Format)
		default:
			return cli.PrintFatalErrorf("Invalid format given: %q", e.Format)
		}
	}

	return ""
}

func (e *EntryUpdate) SetEntryUpdate(cli *Cli) string {
	e.EntryCreate.ApplyEntry(cli)

	// Attribute is required
	if len(e.IDs) == 0 {
		return cli.PrintFatalError("Required positional parameter (ids) is missing")
	}

	// Build a list of entries to update
	entries := make([]*mod.Entry, len(e.IDs))
	for i, id := range e.IDs {
		// Clone entry
		clone := e.EntryCreate.Entry

		// Change ID and add it to the list
		clone.ID = id

		// Omit parameter if not specified on CLI
		if !e.EntryCreate.ParameterSet {
			clone.DontIncludeParametersInRequest()
		}

		entries[i] = &clone
	}

	newEntries, bulkResponse, err := cli.GetApi().PatchEntries(entries)
	if err != nil {
		return cli.PrintFatalError(err.Error())
	}

	switch strings.ToUpper(e.EntryCreate.Format) {
	case "PRETTY", "":
		fmt.Println(bulkResponse.Message.Client)
	case "CSV":
		w := csv.NewWriter(os.Stdout)
		w.Write([]string{
			fmt.Sprintf("%d", bulkResponse.Overview.Successful),
			fmt.Sprintf("%d", bulkResponse.Overview.Errors),
			fmt.Sprintf("%d", bulkResponse.Overview.Exists),
		})
		w.Flush()
	case "JSON":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(struct {
			NewEntries []*mod.Entry                 `json:"new_entries"`
			Response   *mod.BulkResponse[mod.Entry] `json:"response"`
		}{NewEntries: newEntries, Response: bulkResponse})
	default:
		cli.PrintFatalErrorf("Invalid format given: %q", e.EntryCreate.Format)
	}

	return ""
}

func (e *Entry) IsFieldDisabled() bool {
	return e.Disabled
}
