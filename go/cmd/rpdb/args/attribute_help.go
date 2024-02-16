package args

import "git.rpjosh.de/RPJosh/go-logger"

func (a *AttributeList) Help() string {
	return `
Listing of all available attributes.

list      l                  |Shows all available attributes 
    --ids     -i  {id,id}    |Filters the attributes with the given ids
    --name    -n  {xx}       |Only the attribute with the given name will be returned
|___________________________________________________________________________

Global options that can be used for all comamnds.

 --output  {format}  	|Output format to use. Available formats are 'pretty', 'json' and 'csv'
`
}

func (a *Attribute) Help() string {
	return (`
Listing of all available attributes.

list      l                  |Shows all available attributes 

|___________________________________________________________________________

Global options that can be used for all comamnds.

 --output  {format}  	|Output format to use. Available formats are 'pretty', 'json' and 'csv'
    `)
}

func (a *AttributeList) GetOutputFormats(cli *Cli, input string) (rtc []string) {
	return []string{"pretty", "csv", "json"}
}

func (a *AttributeList) GetAttributeNames(cli *Cli, input string) (rtc []string) {
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
