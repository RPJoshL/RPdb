package main

import (
	"github.com/RPJoshL/RPdb/v4/go/api"
	"github.com/RPJoshL/RPdb/v4/go/persistence"
	"git.rpjosh.de/RPJosh/go-logger"
)

const apiKey = "2e24df58317d91d0314fbd98631b6d31a30818a54950c68f8899741025f6a9f0"

func main() {
	// Initialize logger
	configureLogger()
	defer logger.CloseFile()

	// Globally used api options
	apiOptions := api.ApiOptions{}

	// Using the raw api interface without the persistence layer
	api := api.NewApi(apiKey, apiOptions)
	runApi(api)

	// Using the persistence layer with WebSocket support
	pers := persistence.NewPersistence(apiKey, apiOptions, &persistence.PersistenceOptions{
		WebSocket: persistence.WebSocket{UseWebsocket: true},
		Exeuction: *persistence.NewExecution(Execute, ExecuteWithResponse, false),
	})
	runPersistence(pers)
}

// configureLogger configures the gloal logger with some default values
func configureLogger() {
	globalLogger := logger.Logger{
		Level: logger.LevelDebug,
		File: &logger.FileLogger{
			Level: logger.LevelWarning,
		},
		ColoredOutput: true,
		PrintSource:   true,
	}
	logger.SetGlobalLogger(&globalLogger)
}
