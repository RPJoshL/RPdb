package main

import (
	"os"
	"sync"
	"time"

	"github.com/RPJoshL/RPdb/v4/go/client/models"
	mod "github.com/RPJoshL/RPdb/v4/go/models"
	"github.com/RPJoshL/RPdb/v4/go/persistence"
	"git.rpjosh.de/RPJosh/go-logger"
)

// OneShot exits the program if no entries are available within the next
// {duration}
type OneShot struct {
	Duration    time.Duration
	Persistence *persistence.Persistence
	Attributes  *map[int]models.AttributeOptions

	// Mutex to synchronize the os.exit function
	Mtx *sync.Mutex
}

func NewOneShot(duration time.Duration, persistence *persistence.Persistence, attributes *map[int]models.AttributeOptions, execSync *sync.Mutex) *OneShot {
	rtc := &OneShot{
		Duration:    duration,
		Persistence: persistence,
		Attributes:  attributes,
		Mtx:         execSync,
	}

	return rtc
}

// Start starts the scheduling of the OneShot process and logic.
// This method does NOT block
func (o *OneShot) Start(updateChan chan mod.Update) {
	// Initial check
	o.checkAndScheduleOneShot()

	go func() {
		for {
			<-updateChan

			o.checkAndScheduleOneShot()
		}
	}()
}

// checkAndScheduleOneShot validates that an entry in the next x minutes is available
// to execute.
// If no entry is available, the program will stop
func (o *OneShot) checkAndScheduleOneShot() {

	// The maximum allowed execution time for entries
	maxExecutionTime := time.Now().Add(o.Duration)

	// Find the next entry to execute
	for _, e := range o.Persistence.GetEntriesAll() {

		// Only attributes which does have a program registered a counted for one shot
		if attr, doesExist := (*o.Attributes)[e.Attribute.ID]; !doesExist || attr.Program == "" {
			continue
		}

		// If the entry should be executed now, it is ALWAYS valid for one shot
		if e.ShouldExecuteNow() {
			return
		}

		// The execution time has to be in the range of the given one shot time
		if e.DateTimeExecution.Before(maxExecutionTime) {
			logger.Debug("Found entry #%d that is within the time range for one shot", e.ID)
			return
		}
	}

	// Let the executor some time to lock.
	// @TODO how could we make this cleaner?
	time.Sleep(100 * time.Millisecond)

	o.Mtx.Lock()
	logger.Info("Found no entry within the time range of oneShot. Leaving now")
	os.Exit(0)
	o.Mtx.Unlock()
}
