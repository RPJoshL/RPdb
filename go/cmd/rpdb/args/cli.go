package args

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/RPJoshL/RPdb/v4/go/api"
	"github.com/RPJoshL/RPdb/v4/go/client/models"
	mod "github.com/RPJoshL/RPdb/v4/go/models"
	"github.com/RPJoshL/RPdb/v4/go/pkg/cli"
)

// Cli parameters that can be processed without having a concrete app configuration
var AnonymousCliOptions = []string{
	"--version", "-v",
	"--help", "-h", "?",
	"completion", "comp",
}

type Cli struct {

	// This args are passed to the root
	UserConfig     *models.UserConfig     `cli:","`
	RuntimeOptions *models.RuntimeOptions `cli:","`

	// This field is not used! It's only there that the CLI parser won't throw an error
	ConfigPath string `cli:"--config,-conf"`

	Version string `cli:"--version,-v,~~~"`

	// Sub commands
	Entry      *Entry      `cli:"entry,e"`
	Attribute  *Attribute  `cli:"attribute,a"`
	Completion *Completion `cli:"completion,comp"`

	// If the program is called in auto-completion mode
	AutoComplete bool
}

func (cli *Cli) Help() string {
	return (`
Syntax: ProgramName [generic options] entry\|attribute [options]

Generic options (these has to be specified at the beginning and affects only the running program)

  --config        -conf {path}	  |Configuration file path to use|. Defaulting to $CONFIG/RPJosh/RPdb-go/config.yaml
  --multiInstance -mi             |Also notifies the currently used token on updates|. This is required when you are
                                  using the same API-Key multiple times locally (create + listen)
  --quiet         -q              |Instead of a user friendly message the raw data / no date will be printed.

  --service       -s              |Runs this program infinite to execute scheduled entries
  --service-retry -sr             |Automatically retries to fetch data from the server if the initial load fails (no exit)
  --oneShot       -os   {time}    |The program will be exited, when no entries in the next {time} are available.
                                  |The time will be reset after an entry was executed. Example: '3h', '1h10m'
  --version       -v              |Prints the version of the application
|_________________________________________________________________________________________________________

To get a help to the various options, execute these again with the parameter --help.
For example: ProgramName entry --help

  entry      e     |Schedule and manage the execution of entries
  attribute  a     |List all available attributes
  completion comp  |Output shell completion code for the specified shell| (only bash is supproted currently)
	`)
}

func (cli *Cli) EnableAutoComplete() {
	cli.AutoComplete = true
}

func ParseArgs(config *models.AppConfig, args []string) error {
	cl := &Cli{
		UserConfig:     &config.UserConfig,
		RuntimeOptions: &config.RuntimeOptions,
		Entry:          &Entry{},
		Attribute:      &Attribute{},
		Completion:     &Completion{},
	}

	if cli.ParseParams(args, cl) < 0 {
		return fmt.Errorf("")
	}

	return nil
}

// ParseAnonymousArgs is like [ParseArgs] but only processes command line arguments
// that can be handled without a valid configuration file
func ParseAnonymousArgs(args []string) error {
	cl := &Cli{
		UserConfig:     &models.UserConfig{},
		RuntimeOptions: &models.RuntimeOptions{},
		Entry:          &Entry{Disabled: true},
		Attribute:      &Attribute{Disabled: true},
		Completion:     &Completion{},
	}

	if cli.ParseParams(args, cl) < 0 {
		return fmt.Errorf("")
	}

	return nil
}

func (cli *Cli) SetVersion() string {
	fmt.Printf("%s (from %s)\n", mod.LibraryVersion, mod.LibraryVersionDate)
	os.Exit(0)
	return ""
}

// PrintFatalError prints the given message and exits eventually the program
func (cli *Cli) PrintFatalError(message string) string {

	// If the flag '--quiet' is not provided, print the error to stdout
	if !cli.RuntimeOptions.Quiet {

		// Check if coloring should be enabled
		if env, exists := os.LookupEnv("TERMINAL_DISABLE_COLORS"); exists && strings.ToLower(env) == "true" {
			fmt.Fprintln(os.Stderr, message)
		} else {
			fmt.Fprintf(os.Stderr, "\033[1;31m%s\033[0m\n", message)
		}
	}

	// Leave the program when no service or oneShot was given
	if !cli.RuntimeOptions.Service && cli.RuntimeOptions.OneShot == nil {
		os.Exit(1)
	}

	return message
}

// PrintFatalErrorf prints the given message formatted and exits eventually the program
func (cli *Cli) PrintFatalErrorf(message string, params ...any) string {
	return cli.PrintFatalError(fmt.Sprintf(message, params...))
}

// GetApi returns the API interface of this application without the persistence
// layer
func (cli *Cli) GetApi() api.Apiler {
	return api.NewApi(
		cli.UserConfig.ApiKey,
		api.ApiOptions{
			Language:      cli.UserConfig.Langauge,
			MultiInstance: cli.UserConfig.MultiInstance,
			BaseUrl:       cli.UserConfig.BaseURL,
		},
	)
}

func (cli *Cli) PrintStructFormatted(str mod.Formattable, format string) {
	switch strings.ToUpper(format) {
	case "PRETTY", "":
		fmt.Println(str.String())
	case "JSON":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(str)
	case "CSV":
		w := csv.NewWriter(os.Stdout)
		w.Write(str.ToSlice())
		w.Flush()
	default:
		cli.PrintFatalErrorf("Invalid format given: %q", format)
	}
}

func (cli *Cli) PrintStructsFormatted(structs *[]mod.Formattable, format string) {
	switch strings.ToUpper(format) {
	case "PRETTY", "", "CSV":
		for _, a := range *structs {
			cli.PrintStructFormatted(a, format)
		}
	case "JSON":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(structs)
	default:
		cli.PrintFatalErrorf("Invalid format given: %q", format)
	}
}
