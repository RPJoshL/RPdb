package models

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/RPJoshL/RPdb/v4/go/api"
	"github.com/RPJoshL/RPdb/v4/go/persistence"
	"git.rpjosh.de/RPJosh/go-logger"
	yaml "gopkg.in/yaml.v3"
)

// AppConfig is the root configuration struct of the application with
// the various sub configurations
type AppConfig struct {
	UserConfig      UserConfig         `yaml:"user"`
	AttributeConfig []AttributeOptions `yaml:"attributes"`
	LoggerConfig    LoggerConfig       `yaml:"logger"`
	RuntimeOptions  RuntimeOptions
}

// UserConfig contains user specific configuration options like the API key
type UserConfig struct {
	ApiKey        string `yaml:"apiKey"`
	ApiKeyFile    string `yaml:"apiKey_file"`
	Langauge      string `yaml:"language"`
	MultiInstance bool   `yaml:"multiInstance" cli:"--multiInstance,-mi,~~~"`
	BaseURL       string `yaml:"baseURL"`
	SocketURL     string `yaml:"socketURL"`
}

func (c *UserConfig) SetMultiInstance() string {
	c.MultiInstance = true
	return ""
}

// AttributeConfig can contain additional options for a single attribute
type AttributeConfig struct {
	Options []AttributeOptions
}

// AttributeOptions are used to customize the behaviour of a specific attribute
// like defining the execution program or if it should be shown in the UI
type AttributeOptions struct {
	Name              string `yaml:"name"`
	Id                int    `yaml:"id"`
	Hide              bool   `yaml:"hide"`
	Program           string `yaml:"program"`
	OnDeleteProgram   string `yaml:"onDelete"`
	PassOnlyParameter bool   `yaml:"passOnlyParameter"`
}

// LoggerConfig is used to customize the logging output and behaviour
type LoggerConfig struct {
	PrintLevel string `yaml:"printLevel"`
	WriteLevel string `yaml:"logLevel"`
	LogPath    string `yaml:"logPath"`
}

// RuntimeOptions containes options specified via the CLI that are required for
// the further run / while running the application
type RuntimeOptions struct {

	// Runs the program infinite for the abbility to execute programs for the scheduled entries.
	// This will use the persistent layer of the library
	Service bool `cli:"--service,-s,~~~"`

	// Leaves the program when no entries in the next X minutes are available. The time will be reset
	// after an entry was executed
	OneShot *time.Duration `cli:"--oneShot,-os"`

	// Printing raw data instead of a user-friendly message
	Quiet bool `cli:"--quiet,-q,~~~"`
}

func (o *RuntimeOptions) SetService() string {
	o.Service = true
	return ""
}

func (o *RuntimeOptions) SetQuiet() string {
	o.Quiet = true
	return ""
}

func (o *RuntimeOptions) SetOneShot(value string) string {
	// Try to parse the string to a valid time.Duration
	d, err := time.ParseDuration(value)
	if err != nil {
		return "OneShot: " + err.Error()
	}
	o.OneShot = &d

	return ""
}

// GetAppConfig parses the configuration file and applies the CLI parameters afterwards
// through the given function
func GetAppConfig(commandLine bool, configParser func(*AppConfig, []string) error) (*AppConfig, error) {

	// Preparse special command line parameters
	checkHelpAndVersionArgs(configParser)

	// Get the configuration path
	configPath := getConfigPath()
	if configPath == "" {
		return nil, fmt.Errorf("unable to find the location of the configuration file")
	}

	// Parse the configuration
	config := &AppConfig{}
	if err := ParseConfigFile(config, configPath); err != nil {
		return nil, fmt.Errorf("failed to parse the configuration: %s", err)
	}

	// Set default options
	config.SetDefaults()

	// Configure logger
	logg := logger.GetLoggerFromEnv(&logger.Logger{
		Level: logger.GetLevelByName(config.LoggerConfig.PrintLevel),
		File: &logger.FileLogger{
			Level: logger.GetLevelByName(config.LoggerConfig.WriteLevel),
			Path:  config.LoggerConfig.LogPath,
		},
		ColoredOutput: true,
	})
	logger.SetGlobalLogger(logg)

	// Validate app configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %s", err)
	}

	// Parse command line options
	if err := configParser(config, os.Args); err != nil {
		return nil, fmt.Errorf("unable to parse the command line")
	}

	return config, nil
}

// checkHelpAndVersionArgs checks if the first CLI argument is "--version" or
// "--help". For these parameters a configuration file is not needed and are therefore
// parsed manually.
// If one of these parameters were found the program is exited
func checkHelpAndVersionArgs(configParser func(*AppConfig, []string) error) {
	if len(os.Args) <= 1 {
		// No parameters to check
		return
	}

	// Only the first given parameter is checked
	arg := strings.ToLower(os.Args[1])

	// These are all special commands
	if arg == "--version" || arg == "-v" || arg == "--help" || arg == "-h" || arg == "?" {
		// If one of these were found call the configParser function (with empty config)
		configParser(&AppConfig{}, []string{"", arg})
		os.Exit(0)
	}
}

// getConfigPath determines the file location of the configuration file.
// If no matching location could be found, an empty string is returned.
// This function does not validate that the file exists!
func getConfigPath() string {

	// The highest priority has the configuration flag via the CLI parameters
	for i, arg := range os.Args {
		if arg == "-conf" || arg == "--config" && len(os.Args) > i {
			return os.Args[i+1]
		}
	}

	// When no config was given, use the configuration file in the users home directory
	dirName, err := getUsersConfigFile()
	if err != nil {
		return ""
	}

	return dirName
}

// ParseConfigFile parses the given configuration file (.yaml) to an Appconfiguration
func ParseConfigFile(conf *AppConfig, file string) error {
	dat, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(dat, &conf); err != nil {
		return err
	}

	return nil
}

// SetDefaults applies default configuration options if they were
// not set within the configuration file
func (conf *AppConfig) SetDefaults() {

	// Log level
	if conf.LoggerConfig.PrintLevel == "" {
		conf.LoggerConfig.PrintLevel = "info"
	}
	if conf.LoggerConfig.WriteLevel == "" {
		conf.LoggerConfig.WriteLevel = "warning"
	}
}

// Validate validates if this Appconfiguration is valid.
// When an error is found, it will be returned
func (conf *AppConfig) Validate() error {

	// Validate required fields in 'AttributeOptions'
	for _, opt := range conf.AttributeConfig {
		if opt.Name == "" && opt.Id == 0 {
			return fmt.Errorf("for each attribute an id or name is required")
		}
	}

	// Validate and read the JWT key path
	if conf.UserConfig.ApiKeyFile != "" {
		if cnt, err := os.ReadFile(conf.UserConfig.ApiKeyFile); err != nil {
			return fmt.Errorf("failed to read api key from file: %s", err)
		} else if len(string(cnt)) != 64 {
			return fmt.Errorf("got invalid api key from file: %q. The key should be exactly 64 characters long. Got %d", conf.UserConfig.ApiKeyFile, len(string(cnt)))
		} else {
			conf.UserConfig.ApiKey = string(cnt)
		}
	}

	return nil
}

// ToApiOptions is an adapter function to convert this abstract application configuration
// to an api options
func (c *AppConfig) ToApiOptions() api.ApiOptions {
	return api.ApiOptions{
		Language: c.UserConfig.Langauge,
		BaseUrl:  c.UserConfig.BaseURL,
	}
}

// ToWebsocketOptions is an adapter function to convert this abstract application configuration
// to websocket options
func (c *AppConfig) ToWebsocketOptions() persistence.WebSocket {
	return persistence.WebSocket{
		UseWebsocket: true,
		SocketURL:    c.UserConfig.SocketURL,
	}
}
