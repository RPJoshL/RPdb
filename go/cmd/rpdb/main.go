package main

import (
	"context"
	"os"
	"sync"

	"github.com/RPJoshL/RPdb/v4/go/client/models"
	service "github.com/RPJoshL/RPdb/v4/go/client/services"
	"github.com/RPJoshL/RPdb/v4/go/cmd/rpdb/args"
	"github.com/RPJoshL/RPdb/v4/go/persistence"
	"git.rpjosh.de/RPJosh/go-logger"
)

// App contains shared ressource needed für the run of the application
type App struct {
	config   *models.AppConfig
	executor *service.ProgramExecutor

	// Mutex used for oneShot that the program won't be leaved when the program is
	// still executed
	executionSync *sync.Mutex
}

// main provides a simple go application with CLI parameters support
func main() {
	defer logger.CloseFile()

	// Parse and get configuration
	conf, err := models.GetAppConfig(true, args.ParseArgs)
	if err != nil {
		logger.Fatal("Startup failed: %s", err)
	}

	// Nothing to do anymore -> leave
	if conf.RuntimeOptions.OneShot == nil && !conf.RuntimeOptions.Service {
		os.Exit(0)
	}

	// Configure the persistence layer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pers := persistence.NewPersistenceWithContext(
		ctx, conf.UserConfig.ApiKey, conf.ToApiOptions(),
		&persistence.PersistenceOptions{
			WebSocket: conf.ToWebsocketOptions(),
			Exeuction: persistence.Execution{},
		},
	)

	// Initialize the persistence layer
	if err := pers.Start(); err != nil {
		logger.Fatal("Failed to start the persistence layer: %s", err)
	}

	// Index the attribute options by attribute id
	attributeMap := make(map[int]models.AttributeOptions)
	for i, a := range conf.AttributeConfig {

		// Even if an ID is provided directly, we do validate it
		if a.Id != 0 {
			if _, err := pers.GetAttribute(a.Id); err != nil {
				logger.Warning("Unable to get attribute with ID %d: %s", a.Id, err)
			} else {
				attributeMap[a.Id] = conf.AttributeConfig[i]
			}

			continue
		}

		// Try to the the attribute by name
		if attr, err := pers.GetAttributeByName(a.Name); err != nil {
			logger.Warning("Unable to get attribute with name %q: %s", a.Name, err)
		} else {
			attributeMap[attr.ID] = conf.AttributeConfig[i]
		}
	}

	// Create and assign executor
	var mxt sync.Mutex
	app := &App{
		config:        conf,
		executionSync: &mxt,
		executor: &service.ProgramExecutor{
			Attributes: attributeMap,
			Mutex:      &mxt,
		},
	}
	pers.Options.Exeuction.Executor = app.executor.Execute
	pers.Options.Exeuction.ExecuterExecResponse = app.executor.ExecuteResponse

	// Create context which expires in "oneShot" minutes
	if app.config.RuntimeOptions.OneShot != nil {
		oneShot := NewOneShot(*app.config.RuntimeOptions.OneShot, pers, &attributeMap, app.executionSync)

		// Add update hook to persistence
		oneShot.Start(pers.Update.RegisterObserver())
	}

	// Run the program infinite
	select {}
}
