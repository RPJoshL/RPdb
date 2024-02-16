package main

import (
	"time"

	"github.com/RPJoshL/RPdb/v4/go/models"
	"github.com/RPJoshL/RPdb/v4/go/persistence"
	"git.rpjosh.de/RPJosh/go-logger"
)

// runPersistence shows how to use the api WITH the persistence layer in front of
// the api and how to use the WebSocket and Execution function
func runPersistence(pers *persistence.Persistence) {

	// You have to call always start before using the instance
	if err := pers.Start(); err != nil {
		logger.Fatal("Failed to start persistence: %s", pers)
	}

	// Update listener
	updateChan := pers.Update.RegisterObserver()
	go func() {
		for {
			up := <-updateChan
			logger.Info("Received update in observer: %s", up)
			logger.Info("New entries: %s", pers.GetEntriesAll())
		}
	}()

	// Entries with the type "no_db"
	if attrNoDb, err := pers.GetAttributeByName("Keine DB"); err == nil {

		// Create
		ent, err := pers.CreateEntry(models.Entry{Attribute: attrNoDb, Parameters: models.NewParameters("I'm of the type no_db", ""), Offset: "+21m"})
		if err == nil {
			logger.Debug("New entries: %s", pers.GetEntriesAll())
		}

		// Delete
		_, resp, err := pers.DeleteEntries([]int{ent.ID})
		if err == nil {
			logger.Debug("%s. New entries: %s", resp.Message, pers.GetEntriesAll())
		}

	} else {
		logger.Error("Cannot test no_db because attribute was not found")
	}

	// Test WebSocket connection
	if 1 == 1 {
		<-time.After(600 * time.Second)
		logger.Fatal("Leaving now")
	}

	// Get all entries
	logger.Debug("Locally fetched entries: %s", pers.GetEntriesAll())

	// Locally filtered
	ents, err := pers.GetEntries(models.EntryFilter{Parameters: &[]models.NullString{models.NewNullString("What's up here?")}})
	if err == nil {
		logger.Debug("Locally filtered entries: %s", ents)
	}

	// Delete entry
	_, bb, err := pers.DeleteEntries([]int{pers.GetEntriesAll()[1].ID})
	if err == nil {
		logger.Debug("%s: New entries: %s", bb.Message, pers.GetEntriesAll())
	}

	// Create entry
	oo, bc, err := pers.CreateEntries([]*models.Entry{
		{Attribute: pers.GetAttributesAll()[0], Parameters: models.NewParameters("", "Lauter"), DateTime: models.NewDateTime("2026-04-07T20:15:00")},
		{Attribute: pers.GetAttributesAll()[1], Offset: "+21m"},
	})
	if err == nil {
		logger.Debug("%s: New entries: %s", bc.Message, pers.GetEntriesAll())
	}

	// Update entry
	oo[0].Parameters = models.NewParameters("THIS WAS MODIFIED WITH ME", "Lauter")
	rtc, err := pers.UpdateEntry(oo[0])
	if err == nil {
		logger.Debug("%s: New entries: %s", rtc.Message, pers.GetEntriesAll())
	}
}

func Execute(ent models.Entry) {
	logger.Debug("Im received an entry to execute (%d)", ent.ID)
}

func ExecuteWithResponse(ent models.Entry) *models.ExecutionResponse {
	logger.Debug("Oh no. I have to return a response :/")
	return &models.ExecutionResponse{EntryId: ent.ID, Code: 0, Text: "This was successful"}
}
