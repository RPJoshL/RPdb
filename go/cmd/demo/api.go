package main

import (
	"time"

	"github.com/RPJoshL/RPdb/v4/go/api"
	"github.com/RPJoshL/RPdb/v4/go/models"
	"git.rpjosh.de/RPJosh/go-logger"
)

// runApi shows some examples of how to use the raw api interface without
// the persistence layer
func runApi(a *api.Api) {
	// Attribute test //
	attr, err := a.GetAttribute(1)
	if err == nil {
		logger.Debug("Received attribute: %s", *attr)
	}

	attrs, err := a.GetAttributes()
	if err == nil {
		logger.Debug("Received attributes: %s", attrs)
	}

	// Entry test //
	ent, err := a.GetEntry(1954)
	if err == nil {
		logger.Debug("Received entry: %s", ent)
	}

	entries, err := a.GetEntries(models.EntryFilter{})
	if err == nil {
		logger.Debug("Received entries: %s", entries)
	}

	entCrea, err := a.CreateEntry(models.Entry{
		Attribute:  attr,
		Parameters: models.NewParameters("Im not so happy with it...", ""),
		Offset:     "+20m",
	})
	if err == nil {
		logger.Debug("Created entry: %s", entCrea)
	}

	entCrea.Parameters = models.NewParameters("Im NOT not so happy with it...", "")
	entUpdate, err := a.UpdateEntry(entCrea)
	if err == nil {
		logger.Debug("Updated entry: %s", entUpdate)
	}

	rspMessage, err := a.DeleteEntry(entUpdate.ID)
	if err == nil {
		logger.Debug("%s", rspMessage.String())
	}

	bulkCreate, _, err := a.CreateEntries([]*models.Entry{
		{Attribute: attr, Parameters: []models.EntryParameter{{Preset: "Lauter"}}, DateTime: models.NewDateTime("2025-04-07T19:15:00")},
		{Attribute: attr, Parameters: []models.EntryParameter{{Value: "I'm here"}}, Offset: "+21m"},
	})
	if err == nil {
		logger.Debug("Created entries: %s", bulkCreate)
	}

	bulkCreate[0].Parameters = []models.EntryParameter{{Value: "I AM THE HERO"}}
	bulkUpdate, _, err := a.UpdateEntries([]*models.Entry{
		bulkCreate[0],
	})
	if err == nil {
		logger.Debug("Updated entries: %s", bulkUpdate)
	}

	bulkDelete, bb, err := a.DeleteEntries([]int{bulkCreate[0].ID})
	if err == nil {
		logger.Debug("%s: %d", bb.Message, bulkDelete)
	}

	filterDelete, err := a.DeleteEntriesFiltered(models.EntryFilter{
		Parameters: &[]models.NullString{models.NewNullString("I'm here")},
		//Attributes: []int{attr.ID},
	})
	if err == nil {
		logger.Debug("%s: %d", filterDelete.Message, filterDelete.IDs)
	}

	// Updates //
	upd, err := a.GetUpdate(api.UpdateRequest{LaterThan: time.Now().Add(-5 * time.Second)})
	if err == nil {
		logger.Debug("Received update: %s", upd)
	}
}
