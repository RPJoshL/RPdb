package persistence

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RPJoshL/RPdb/v4/go/api"
	"github.com/RPJoshL/RPdb/v4/go/models"
	"github.com/RPJoshL/RPdb/v4/go/pkg/utils"
	"git.rpjosh.de/RPJosh/go-logger"
)

// ExecutionType states the type of execution for which the
// entry should be executed
type ExecutionType int

const (
	// Default execution behaviour when the ExecutionTime of the Entry has been reached
	DEFAULT ExecutionType = iota

	// Delete is called for an Entry in the past (NOW < ExecutionTime) that was deleted from the user.
	// It's referenced in RPdb as an "onDeleteHook"
	DELETE
)

// Execution manages the scheduling of entries and calls your custom
// function on execution.
//
// This is also responsible to trigger an update when an old entry
// is removed from the locally cached list
type Execution struct {

	// Function that is called when an entry should be executed
	Executor func(models.Entry, ExecutionType)

	// Function that is called when an entry of the type "exec_response"
	// was executed.
	// You have to return an execution response or nil for no response
	ExecuterExecResponse func(models.Entry) *models.ExecutionResponse

	// By default, an entry is kept in the locale list until the date fields
	// "DateTime" and "DateTimeExecution" are past.
	//
	// When this field is set the "DateTimeExecution" field will be ignored
	// and the entries are immediately removed from the list
	IgnoreExecutionTime bool

	// By default, an update is only triggered if the DatetimeExecution AND
	// the DateTime are in the past.
	// With this option an (empty) update will also be fired when the DateTime
	// is overwritten
	TriggerUpdateOnDateTimeChanges bool

	// Managed by persistence: update struct for tiggering updates
	Update *PersistenceUpdate

	// Managed by persistence: API interface to get and delete the entries from
	Api api.Apiler

	// Managed by persistence: base context to use for scheduling
	BaseContext context.Context

	// Persitence entry to remove the entries from
	persEntry *persistenceEntry

	// The context of the currently scheduled executions
	context       context.Context
	cancelContext context.CancelFunc

	// Mutex to synchronize cancel function and context access
	mtx sync.Mutex

	// Timer for removing the entry from the list
	normalTimer *time.Timer

	// The ID of the entry to execute next
	nextEntry atomic.Int64
}

// NewExecution creates a new struct for scheduling the execution of entries.
// It does call the given function on execution and removes old entries from local
// cache
func NewExecution(executor func(models.Entry, ExecutionType), executerExecResponse func(models.Entry) *models.ExecutionResponse, ignoreExecutionTime bool) *Execution {
	return &Execution{
		Executor:             executor,
		ExecuterExecResponse: executerExecResponse,
		IgnoreExecutionTime:  ignoreExecutionTime,
	}
}

// StartScheduling starts the scheduling of the executions.
// If an entry was executed it will be removed from the local list and
// the "Executor()" function with a copy of the entry will be called.
// After that the scheduling will be resetted for the next entry
func (e *Execution) StartScheduling() {
	e.mtx.Lock()

	// Cancel contexts
	if e.cancelContext != nil {
		e.cancelContext()
	}
	// Create a new context
	e.context, e.cancelContext = context.WithCancel(e.BaseContext)

	// Stop old timers
	if e.normalTimer != nil {
		e.normalTimer.Stop()
	} else {
		// Start a "fake" timer which will fier in 85 days
		e.normalTimer = time.NewTimer(85 * 365 * time.Hour)
	}

	// Start a channel which is listening for the timers event
	go func() {
		for {
			select {
			case <-e.normalTimer.C:
				e.handleExecution()
			case <-e.context.Done():
				return
			}
		}
	}()
	e.mtx.Unlock()

	e.schedule()
}

// schedule schedules the next execution of the entries and stops all
// old timers
func (e *Execution) schedule() {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	// Get the next entry to execute
	nextEntry := e.getNextEntryNormal(nil)

	// Start timer
	if nextEntry != nil {
		// Update the next ID
		e.nextEntry.Store(int64(nextEntry.ID))

		// Get the date on which the timer should fire
		dateTime := nextEntry.DateTime.Time
		if !nextEntry.WasExecuted() && !nextEntry.DateTimeExecution.IsZero() && nextEntry.DateTimeExecution.Before(dateTime) && nextEntry.DateTimeExecution.After(time.Now()) {
			dateTime = nextEntry.DateTimeExecution.Time
		}

		logger.Debug(utils.Sprintfl("Scheduled next execution in %.1f seconds (#%d)", time.Until(dateTime).Seconds(), nextEntry.ID))

		if e.normalTimer == nil {
			return
		} else {
			e.normalTimer.Stop()
			e.normalTimer.Reset(time.Until(dateTime))
		}
	} else {
		// Reset the times
		logger.Debug("Clearing timer for execution")
		e.normalTimer.Stop()
		e.nextEntry.Store(0)
	}
}

// handleExecution handles the immediate execution of the next entry
func (e *Execution) handleExecution() {
	e.mtx.Lock()

	// Get the entry to execute next
	nextEntryId := e.nextEntry.Load()
	if nextEntryId == 0 {
		logger.Warning("Should execute entry now but couldn't determine the next entry")
		e.mtx.Unlock()
		return
	}

	nextEntry, _ := e.Api.GetEntry(int(nextEntryId))
	if nextEntry == nil {
		logger.Warning("Should execute entry now but couldn't find an entry with id %d", nextEntryId)
		e.mtx.Unlock()
		return
	}

	e.mtx.Unlock()

	// Check weather to execute the entry or just trigger an update
	if nextEntry.ShouldExecuteNow() {
		e.Execute(nextEntry)
	}

	// The entry can be removed because the dates of "DateTime" and
	// "DateTimeExecution" are passed
	if nextEntry.IsPast(e.IgnoreExecutionTime) {
		// The entry will be removed within the next reschedule
		e.schedule()
	} else if e.TriggerUpdateOnDateTimeChanges && !nextEntry.ShouldExecuteNow() {
		logger.Debug("Triggering an update that the entries DateTime is past")
		// A rescheduling is not needed because reschedule is triggered from outside
		e.Update.notifyForUpdates(nil)
	} else {
		// Try to schedule the next entry
		e.schedule()
	}
}

// getNextEntryNormal returns the entry that should be executed
// at the next time. If no entry was found nil will be returned.
// If any old entries are found they got removed / executed immediately.
//
// Because this function can call itself recursively you have to provide nil or
// the previous update object for the update.
// In the last function call all previously removed entries will be handled
func (e *Execution) getNextEntryNormal(update *models.UpdateData[*models.Entry]) (rtc *models.Entry) {
	e.persEntry.mux.RLock()

	// First call of the function
	if update == nil {
		update = &models.UpdateData[*models.Entry]{}
	}

	for i := range e.persEntry.data {
		if e.persEntry.data[i].ShouldExecuteNow() {
			// Execute the entry immediate
			e.Execute(e.persEntry.data[i])

			// The entry should be removed if it's dateTime is in the past
			if e.persEntry.data[i].IsPast(e.IgnoreExecutionTime) {
				// Mark it for removal
				update.Deleted = append(update.Deleted, e.persEntry.data[i].ID)
			} else {
				// It could be possible that this entry should be executed next.
				// So the rescheduling has to be done again from the beginning
				e.persEntry.mux.RUnlock()
				return e.getNextEntryNormal(update)
			}
		} else if e.persEntry.data[i].IsPast(e.IgnoreExecutionTime) {
			// Mark it for removal
			update.Deleted = append(update.Deleted, e.persEntry.data[i].ID)
		} else if rtc == nil ||
			// Check if the execution time is before rtc and both were not already executed
			(e.persEntry.data[i].GetExecutionTime(e.IgnoreExecutionTime).Before(rtc.GetExecutionTime(e.IgnoreExecutionTime)) && !e.persEntry.data[i].WasExecuted()) && !rtc.WasExecuted() ||
			// If rtc was scheduled for DateTime (already executed) and this DateTimeExecution is less than rtc's DateTime
			(rtc.WasExecuted() && !e.persEntry.data[i].WasExecuted() && e.persEntry.data[i].GetExecutionTime(e.IgnoreExecutionTime).Before(rtc.DateTime.Time)) ||
			// Check also if the normal date is before rtc's execution time
			e.persEntry.data[i].DateTime.Time.Before(rtc.GetExecutionTime(e.IgnoreExecutionTime)) ||
			// And finally check if the normal date is before rtc's normal time
			e.persEntry.data[i].DateTime.Time.Before(rtc.DateTime.Time) {
			// We finally found an entry which execution time or dateTime is before rtc, and it is not in the past
			rtc = e.persEntry.data[i]
		}
	}
	e.persEntry.mux.RUnlock()

	// Notify for updates if an entry was deleted or removed
	if len(update.Deleted) > 0 {
		e.persEntry.handleUpdate(*update)
		e.Update.notifyForUpdates(models.NewUpdateWithData(update.Deleted, update.Updated, update.Created))

		// Return nil because update calls this function again
		return nil
	}

	return
}

// Execute executes the given entry and marks the entry as executed
// if the attribute is from the type "exec_response"
func (e *Execution) Execute(ent *models.Entry) {
	logger.Debug("Executing entry %s with attribute %q (#%d)", ent.DateTime.FormatPretty(), ent.Attribute.Name, ent.ID)

	// Mark entry as exeucted (locally and also in the api for EA)
	ent.SetExecuted(true)
	if ent.Attribute.ExecuteAlways {
		go func(id int) {
			if err := e.Api.MarkEntryAsExecuted(id); err != nil {
				logger.Warning("Failed to register entry %d as executed: %s", id, err)
			}
		}(ent.ID)
	}

	// Call the execute function
	if e.Executor != nil {
		go func(ent models.Entry) {
			e.Executor(ent, DEFAULT)
		}(*ent)
	}
}

func (e *Execution) ExecuteDelete(ent *models.Entry) {
	logger.Debug("Executing delete hook for entry %s with attribute %q (#%d)", ent.DateTime.FormatPretty(), ent.Attribute.Name, ent.ID)

	// Call the execute function
	if e.Executor != nil {
		go func(ent models.Entry) {
			e.Executor(ent, DELETE)
		}(*ent)
	}
}

// ExecuteExecResponse executes an entry with an attribute of the
// type "exec_response" and returns the execution response. This method
// does block until a response was received
//
// If no function was provided to execute this entry nil is returned as
// a response.
func (e *Execution) ExecuteExecResponse(ent *models.Entry) *models.ExecutionResponse {
	if e.ExecuterExecResponse == nil {
		return nil
	} else {
		return e.ExecuterExecResponse(*ent)
	}
}
