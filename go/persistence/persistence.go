// persistence adds a layer on top of the API to persist the
// data used within the application.
//
// It does also implement functions to keep the persisted data
// up to date and adds an execution support
package persistence

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/RPJoshL/RPdb/v4/go/api"
	"git.rpjosh.de/RPJosh/go-logger"
)

// Persistence is a wrapper around the API interface with additional
// functions to store the fetched data locally within the application.
//
// If you make for example multiple calls to "GetEntries()" the data
// is not fetched from the API every time, but either returned from the locale
// persistence layer (if already available).
//
// You can use the function "NewPersistence()" for creating an instance of this.
// The configuration of other parts like the WebSocket und execution are configured
// with "PersistenceOptions"
type Persistence struct {
	api.Api

	Options *PersistenceOptions

	/* Persistence data struct */

	entry     persistenceEntry
	attribute persistenceAttribute

	// Information to handle an update of the locally cached data
	Update *PersistenceUpdate

	// Base context for all operations
	context context.Context
}

// PersistenceOptions contains options for various modules of the persistence layer
// like the options for the WebSocket or the Execution
type PersistenceOptions struct {
	WebSocket WebSocket

	Exeuction *Execution

	// Function to call before triggering an update after a full reload of the
	// data (or after the initial trough of the [Start] function)
	BeforeInitialUpdateRequest func(p *Persistence)
}

// NewPersistence creates a new persistence layer based on the given API.
// To finish the creation you have to call "Start()".
func NewPersistence(apiKey string, apiOptions api.ApiOptions, persistenceOptions *PersistenceOptions) *Persistence {
	return NewPersistenceWithContext(context.Background(), apiKey, apiOptions, persistenceOptions)
}

// NewPersistenceWithContext creates a new persistence layout based on the given API.
// To finish the creation you have to call "Start()".
func NewPersistenceWithContext(context context.Context, apiKey string, apiOptions api.ApiOptions, persistenceOptions *PersistenceOptions) *Persistence {
	// Don't resolve attributes because they are cached locally
	apiOptions.TreatAsJavaClient = true

	pers := &Persistence{
		Api:     *api.NewApiWithContext(context, apiKey, apiOptions),
		Options: persistenceOptions,
		Update:  &PersistenceUpdate{},
		context: context,
	}

	// Set default values for persistence options
	pers.Options.WebSocket.ApiKey = apiKey
	pers.Options.WebSocket.BaseContext = context
	pers.Options.WebSocket.OnMessage = pers.handleWebSocketMessage
	pers.Options.WebSocket.Update = pers.Update
	if pers.Options.WebSocket.SocketURL == "" {
		pers.Options.WebSocket.SocketURL = "wss://rpdb.rpjosh.de/api/v1/socket"
	}

	// Create persistence data layout for every entity
	pers.entry = persistenceEntry{api: pers}
	pers.attribute = persistenceAttribute{api: pers}

	// Initialize executor
	pers.Options.Exeuction.BaseContext = context
	pers.Options.Exeuction.Api = pers
	pers.Options.Exeuction.Update = pers.Update
	pers.Options.Exeuction.persEntry = &pers.entry

	return pers
}

// Start boots up the persistence layer and makes it ready
// for further usage. This method does block, because it makes
// requests against the api.
//
// Don't ever use any of the functions provided by the persistence layer
// without calling this "Start()" function first. If you do so it's YOUR fault
func (p *Persistence) Start() error {

	// Try to laod the data
	loadError := p.ReloadData()
	if loadError != nil {
		return loadError
	}

	// Start the executor listen for updates
	p.Options.Exeuction.StartScheduling()
	executionUpdateChannel := p.Update.RegisterObserver()
	go func() {
		for {
			select {
			case <-executionUpdateChannel:
				p.Options.Exeuction.schedule()
			case <-p.context.Done():
				logger.Debug("Aborted to listen for updates (execution)")
				return
			}
		}
	}()

	// Start WebSocket
	p.Options.WebSocket.Start()

	return nil
}

// ReloadData forces a full reload of the persisted
// data.
// Locally fetched entries with the flag 'no_db' are
// currently getting lost
func (p *Persistence) ReloadData() error {
	var errEnt error
	var errAttr error

	// The time when the data got fetched
	timeFetch := time.Now()

	var wg sync.WaitGroup
	wg.Add(2)

	// Load the data
	go func() {
		errEnt = p.entry.loadData()
		wg.Done()
	}()
	go func() {
		errAttr = p.attribute.loadData()
		wg.Done()
	}()
	wg.Wait()

	// Return error if one occures
	if errAttr != nil {
		return fmt.Errorf("failed to load attributes: %s", errAttr)
	} else if errEnt != nil {
		return fmt.Errorf("failed to load entries: %s", errEnt)
	}

	// No error occurred. Set the attribute references for the entries (they were not fetched to save bandwidth)
	p.entry.mux.Lock()
	p.entry.linkAttributes(&p.entry.data)
	p.entry.mux.Unlock()

	// Set the last fetch time
	p.Update.versionLock.Lock()
	p.Update.VersionDate = timeFetch
	p.Update.versionLock.Unlock()

	// Trigger update after first load
	if p.Options.BeforeInitialUpdateRequest != nil {
		p.Options.BeforeInitialUpdateRequest(p)
	}
	p.Update.notifyForUpdates(nil)

	return nil
}
