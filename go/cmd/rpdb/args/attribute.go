package args

import (
	"strconv"
	"strings"

	mod "github.com/RPJoshL/RPdb/v4/go/models"
)

// Attribute contains attribute options for the CLI
type Attribute struct {
	Disabled      bool
	AttributeList AttributeList `cli:"list,l"`
}

type AttributeList struct {
	IDs    string `cli:"--ids,-i"`
	Name   string `cli:"--name,-n" completion:"GetAttributeNames"`
	Format string `cli:"--output,-o" completion:"GetOutputFormats"`
}

// SetAttributeList lists all available attributes filtered by the specified fields
func (al *AttributeList) SetAttributeList(cli *Cli) string {

	// Validate input
	if al.Name != "" && al.IDs != "" {
		return cli.PrintFatalError("The arguments '--ids' and '--name' cannot be used together")
	}

	// Parse IDs
	var ids []int
	if al.IDs != "" {
		for i, val := range strings.Split(al.IDs, ",") {
			if intVal, err := strconv.Atoi(val); err != nil {
				return cli.PrintFatalErrorf("Invalid number given for '--ids' at position %d: %q", i, val)
			} else {
				ids = append(ids, intVal)
			}
		}
	}

	// Make the request (get all attributes). The filtering is currently only done locally (there shouldn't be much attributes :)
	attributes, err := cli.GetApi().GetAttributes()
	if err != nil {
		return cli.PrintFatalError(err.Error())
	}

	// Filter the attributes
	var rtc []mod.Formattable

	for _, a := range attributes {

		// Name does not match
		if al.Name != "" && al.Name != a.Name {
			continue
		}

		// ID must be contained in the list
		if len(ids) > 0 {
			found := false
			for _, id := range ids {
				if a.ID == id {
					found = true
					break
				}
			}

			if !found {
				continue
			}
		}

		rtc = append(rtc, a)
	}

	// If only one ID was given, don't return an array (relevant for JSON)
	if len(ids) == 1 {
		if len(rtc) == 0 {
			cli.PrintStructFormatted(nil, al.Format)
		} else {
			cli.PrintStructFormatted(rtc[0], al.Format)
		}
		return ""
	}

	cli.PrintStructsFormatted(&rtc, al.Format)
	return ""
}

func (al *Attribute) IsFieldDisabled() bool {
	return al.Disabled
}
