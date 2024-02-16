package persistence

import (
	"sync"
	"time"

	"github.com/RPJoshL/RPdb/v4/go/models"
	"git.rpjosh.de/RPJosh/go-logger"
)

// PersistenceUpdate contains all needed information about the current
// version of the program to process updates
type PersistenceUpdate struct {
	// The current version number of the data
	Version int
	// The date of the last version
	VersionDate time.Time

	versionLock sync.RWMutex

	// All observers of the update chanel
	observers    []chan models.Update
	observerLock sync.RWMutex
}

// handleWebSocketMessage is the entry point to processes received message from the WebSocket
func (p *Persistence) handleWebSocketMessage(msg models.WebSocketMessage) {

	if msg.Type == models.WebSocketTypeUpdate {
		// A new update of the data was received
		logger.Debug("Received update: %s", msg.Update)

		// Update version information
		p.Update.versionLock.Lock()
		p.Update.Version = msg.Update.Version
		p.Update.VersionDate = msg.Update.VersionDate.Time
		p.Update.versionLock.Unlock()

		// Merge the update
		if msg.Update.Attribute.IsUpdate() {
			p.attribute.handleUpdate(msg.Update.Attribute)
		}
		if msg.Update.Entry.IsUpdate() {
			p.entry.handleUpdate(msg.Update.Entry)
		}

		// Trigger update if something was changed (socket open message may contain no update)
		if msg.Update.Entry.IsUpdate() || msg.Update.Attribute.IsUpdate() {
			p.Update.notifyForUpdates(&msg.Update)
		}
	} else if msg.Type == models.WebSocketTypeExecResponse {
		p.entry.linkAttribute(&msg.ExecResponse)
		resp := p.Options.Exeuction.ExecuteExecResponse(&msg.ExecResponse)
		if resp != nil {
			p.Options.WebSocket.SendExecutionResponse(*resp)
		}
	} else if msg.Type == models.WebSocketTypeNoDb {
		// Link attributes and add to the list
		p.entry.linkAttributes(&msg.NoDb)
		p.entry.addAndSort(msg.NoDb...)

		// Trigger update
		p.Update.notifyForUpdates(&msg.Update)
	}
}

// notifyForUpdates notifies all observer for an update.
// The update can be nil if no update information is available
// (initial loading of the data)
func (p *PersistenceUpdate) notifyForUpdates(update *models.Update) {
	p.observerLock.RLock()
	defer p.observerLock.RUnlock()

	for _, obs := range p.observers {
		go func(c chan models.Update) {
			// The update is not passed by reference that the update information
			// cannot be modified. The data inside the update struct are still
			// passed by reference (pointers)
			c <- *update
		}(obs)
	}
}

// RegisterObserver returns a new channel that is filled when an update
// of the data occur.
// You can check the models.Update methods to get more exact update details.
// Note that the models.Update can also be empt (.IsZero()) after the first
// initial loading. In such a case the entries and attributes were "updated"
func (p *PersistenceUpdate) RegisterObserver() chan models.Update {
	p.observerLock.Lock()
	defer p.observerLock.Unlock()

	c := make(chan models.Update)
	p.observers = append(p.observers, c)
	return c
}

// RemoveObserver removes the given observer from the internal observers
// lists and closed the channel
func (p *PersistenceUpdate) RemoveObserver(c chan models.Update) {
	p.observerLock.Lock()
	defer p.observerLock.Unlock()

	// Find the observer and remove it
	for i := range p.observers {
		if p.observers[i] == c {
			p.observers = append(p.observers[:i], p.observers[i+1:]...)
			close(c)
			break
		}
	}
}
