package main

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/RPJoshL/RPdb/v4/go/client/models"
	service "github.com/RPJoshL/RPdb/v4/go/client/services"
	"github.com/RPJoshL/RPdb/v4/go/cmd/rpdb/args"
	"github.com/RPJoshL/RPdb/v4/go/persistence"
	"git.rpjosh.de/RPJosh/go-logger"
)

// App contains shared ressource needed for the run of the application
type App struct {
	config   *models.AppConfig
	executor *service.ProgramExecutor

	// Mutex used for oneShot so the program won't be leaved when the program is
	// still executed
	executionSync *sync.Mutex

	// Fetched attribute configuration from the config indexed by the ID
	attributeMap map[int]models.AttributeOptions
}

// main provides a simple go application with CLI parameters support
func main() {
	defer logger.CloseFile()

	// Parse and get configuration
	conf, err := models.GetAppConfig(true, args.ParseArgs)
	if err != nil {
		if err != models.ErrCliParse {
			CheckForAnonymousArgs(args.AnonymousCliOptions)
		}

		logger.Fatal("Startup failed: %s", err)
	}

	// Nothing to do anymore -> leave
	if conf.RuntimeOptions.OneShot == nil && !conf.RuntimeOptions.Service && !conf.RuntimeOptions.ServiceRetry {
		os.Exit(0)
	}

	// Assign app variables
	app := &App{
		config:        conf,
		executionSync: &sync.Mutex{},
		attributeMap:  make(map[int]models.AttributeOptions),
	}

	// Configure the persistence layer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pers := persistence.NewPersistenceWithContext(
		ctx, conf.UserConfig.ApiKey, conf.ToApiOptions(),
		&persistence.PersistenceOptions{
			WebSocket:                  conf.ToWebsocketOptions(),
			Exeuction:                  &persistence.Execution{},
			BeforeInitialUpdateRequest: app.initExecutor,
		},
	)

	// Initialize the persistence layer
	StartPersistence(pers, conf.RuntimeOptions.ServiceRetry, 0)

	// Create context which expires in "oneShot" minutes
	if app.config.RuntimeOptions.OneShot != nil {
		oneShot := NewOneShot(*app.config.RuntimeOptions.OneShot, pers, &app.attributeMap, app.executionSync)

		// Add update hook to persistence
		oneShot.Start(pers.Update.RegisterObserver())
	}

	// Run the program infinite
	select {}
}

// initExecutor initializes the executor after the persistence data were loaded
// and maps the attribute config to the correct attribute
func (app *App) initExecutor(pers *persistence.Persistence) {
	for i, a := range app.config.AttributeConfig {

		// Even if an ID is provided directly, we do validate it
		if a.Id != 0 {
			if _, err := pers.GetAttribute(a.Id); err != nil {
				logger.Warning("Unable to get attribute with ID %d: %s", a.Id, err)
			} else {
				app.attributeMap[a.Id] = app.config.AttributeConfig[i]
			}

			continue
		}

		// Try to the the attribute by name
		if attr, err := pers.GetAttributeByName(a.Name); err != nil {
			logger.Warning("Unable to get attribute with name %q: %s", a.Name, err)
		} else {
			app.attributeMap[attr.ID] = app.config.AttributeConfig[i]
		}
	}

	// Init executor
	app.executor = &service.ProgramExecutor{
		Attributes: app.attributeMap,
		Mutex:      app.executionSync,
	}

	// Assign exeuctor to persistence
	pers.Options.Exeuction.Executor = app.executor.Execute
	pers.Options.Exeuction.ExecuterExecResponse = app.executor.ExecuteResponse
}

// CheckForAnonymousArgs checks if the first CLI argument is whitelisted to be used "anonymously" without
// a valid configuration file. They are parsed manually inside this function.
// If one of these parameters were found, the program is exited
func CheckForAnonymousArgs(anonymousArgs []string) {
	if len(os.Args) <= 1 {
		// No parameters to check
		return
	}

	// Only the first given parameter is checked
	arg := strings.ToLower(os.Args[1])
	for _, anonArg := range anonymousArgs {
		if anonArg == arg {
			e := args.ParseAnonymousArgs(os.Args)
			if e != nil {
				logger.Fatal("Failed to parse CLI args: %s", e)
			}
			os.Exit(0)
		}
	}
}

// StartPersistence tries to load all required data for the initial startup
func StartPersistence(pers *persistence.Persistence, retry bool, retryCounter int) {
	if err := pers.Start(); err != nil {
		if retry {
			waitTime := persistence.GetReconnectTimeout(retryCounter + 1)
			logger.Info("Starting of the persistence layer failed. Trying again in %.0f seconds", waitTime.Seconds())
			time.Sleep(waitTime)
			StartPersistence(pers, retry, retryCounter+1)
		} else {
			logger.Fatal("Failed to start the persistence layer: %s", err)
		}
	}
}
