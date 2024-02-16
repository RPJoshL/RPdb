package persistence

import (
	"fmt"
	"sort"
	"sync"

	"github.com/RPJoshL/RPdb/v4/go/api"
	"github.com/RPJoshL/RPdb/v4/go/models"
	"github.com/RPJoshL/RPdb/v4/go/pkg/utils"
	"git.rpjosh.de/RPJosh/go-logger"
)

type persistenceEntry struct {
	// API interface to load the data from (Persistence)
	api api.Apiler

	data []*models.Entry

	// Mutex to synchronize the access to the data
	mux sync.RWMutex
}

// loadData fetches all attributes from the API and stores it locally.
// You have to link the attributes after this operation manually
func (p *persistenceEntry) loadData() error {
	ent, err := p.api.GetRealApi().GetEntries(models.EntryFilter{})
	if err != nil {
		return err
	}

	// Update locally stored data by replacing the value
	p.mux.Lock()
	p.data = ent
	p.mux.Unlock()

	return nil
}

// linkAttributes links the attributes of the given entries to the locally
// fetched attributes
func (p *persistenceEntry) linkAttributes(entries *[]*models.Entry) {
	// Loop through all entries and assign attribute
	for i := range *entries {
		p.linkAttribute((*entries)[i])
	}
}

// linkAttribute links the attribute of the given entry to the locally
// fetched attribute
func (p *persistenceEntry) linkAttribute(entry *models.Entry) {
	if attr, err := p.api.GetAttribute(entry.Attribute.ID); err == nil {
		entry.Attribute = attr
	} else {
		logger.Error("Failed to find attribute with id %d for entry %d", entry.Attribute.ID, entry.ID)
	}
}

// addAndSort adds all the given entries to the local cache and sorts the whole
// array again.
// This method does lock the data mutex
func (p *persistenceEntry) addAndSort(entries ...*models.Entry) {
	p.mux.Lock()
	p.addAndSortWithoutLock(entries...)
	p.mux.Unlock()
}

// addAndSortWithoutLock adds all the given entries to the local cache and sorts the whole
// array again.
// This method does NOT lock the data mutex
func (p *persistenceEntry) addAndSortWithoutLock(entries ...*models.Entry) {
	p.data = append(p.data, entries...)
	sort.SliceStable(p.data, func(i, j int) bool {
		return p.data[i].DateTime.Compare(p.data[j].DateTime.Time) == -1
	})
}

func (p *Persistence) GetEntry(id int) (*models.Entry, *models.ErrorResponse) {
	p.entry.mux.RLocker().Lock()
	defer p.entry.mux.RLocker().Unlock()

	for i, e := range p.entry.data {
		if (*e).ID == id {
			return p.entry.data[i], nil
		}
	}

	return nil, &models.ErrorResponse{ID: "ENTRY_NOT_FOUND", ResponseCode: 404, Message: "Entry was not found"}
}

func (p *Persistence) GetEntries(filter models.EntryFilter) (rtc []*models.Entry, err *models.ErrorResponse) {

	// No filter condition means that all entries should be returned
	if filter.IsZero() {
		p.entry.mux.RLocker().Lock()
		// Entry is copied during reassignment
		rtc = p.entry.data
		p.entry.mux.RLocker().Unlock()
		return
	}

	// The filtering can be applied on the client side with no additional
	// api call
	if filter.CanHandleLocally() && len(filter.Executed) == 0 {
		p.entry.mux.RLocker().Lock()
		for i, e := range p.entry.data {
			if filter.DoesMatch(*e) {
				rtc = append(rtc, p.entry.data[i])
			}
		}
		p.entry.mux.RUnlock()
		return
	}

	// The filtering can not be executed locally so an additional api call is required
	rtc, err = p.Api.GetEntries(filter)
	if err != nil {
		p.entry.linkAttributes(&rtc)
	}
	return
}

// GetEntriesAll is the same function as "GetEntries()" without
// any filter condition
func (p *Persistence) GetEntriesAll() []*models.Entry {
	// No error can occure while fetching the entries locally
	ent, _ := p.GetEntries(models.EntryFilter{})

	return ent
}

func (p *Persistence) DeleteEntry(id int) (resp *models.ResponseMessageWrapper, err *models.ErrorResponse) {
	// Only call api for an entry that is not of the type no_db
	if ent, err2 := p.GetEntry(id); err2 == nil || ent == nil || !ent.Attribute.NoDb {
		resp, err = p.Api.DeleteEntry(id)
	}

	if err == nil {
		// Remove the entry from the locale storage
		p.entry.mux.Lock()
		defer p.entry.mux.Unlock()

		for i, e := range p.entry.data {
			if e.ID == id {
				utils.Remove(&p.entry.data, i)

				// Notify for updates
				p.Update.notifyForUpdates(models.NewUpdateWithData([]int{id}, []*models.Entry{}, []*models.Entry{}))
				return resp, err
			}
		}

		logger.Debug("No entry found to remove with id %d", id)
		return resp, err
	} else {
		return resp, err
	}
}

func (p *Persistence) DeleteEntries(idsToDelete []int) (deleted []int, resp *models.BulkResponse[int], err *models.ErrorResponse) {

	// Filter entries that are of the type no_db
	entriesNoDb := make([]int, 0)
	for i, id := range idsToDelete {
		if ent, err := p.GetEntry(id); err == nil && ent != nil && ent.Attribute.NoDb {
			// Add it to the list of no_db and remove it from api deletion
			entriesNoDb = append(entriesNoDb, id)
			idsToDelete = utils.Remove(&idsToDelete, i)
		}
	}

	// Execute the api request
	if len(idsToDelete) > 0 {
		deleted, resp, err = p.Api.DeleteEntries(idsToDelete)
	} else {
		// Add a response message (@TODO translate)
		resp = &models.BulkResponse[int]{Message: models.ResponseMessage{Client: fmt.Sprintf("All entries were successfully deleted (%d)", len(entriesNoDb))}}
	}

	// Append the no_db data to the list of the deleted entries (if any)
	if err == nil && len(entriesNoDb) > 0 {
		deleted = append(deleted, entriesNoDb...)
		for _, id := range entriesNoDb {
			resp.ResponseData = append(resp.ResponseData, models.BulkResponseData[int]{Status: models.StatusDeleted, StatusCode: 200, Data: id})
		}
	}

	if err == nil && len(deleted) > 0 {
		deletedCopy := deleted
		p.entry.mux.Lock()
		utils.Filter(&deletedCopy, &p.entry.data, func(a int, b *models.Entry) bool { return a == b.ID })
		p.entry.mux.Unlock()

		// Notify for updates
		p.Update.notifyForUpdates(models.NewUpdateWithData(deleted, []*models.Entry{}, []*models.Entry{}))
	}

	return deleted, resp, err
}
func (p *Persistence) DeleteEntriesFiltered(filter models.EntryFilter) (api.EntryDeleteFiltered, *models.ErrorResponse) {
	deleted, err := p.Api.DeleteEntriesFiltered(filter)
	if err == nil {
		deletedCopy := deleted.IDs
		p.entry.mux.Lock()
		utils.Filter(&deletedCopy, &p.entry.data, func(a int, b *models.Entry) bool { return a == b.ID })
		p.entry.mux.Unlock()

		// Notify for updates
		p.Update.notifyForUpdates(models.NewUpdateWithData(deleted.IDs, []*models.Entry{}, []*models.Entry{}))
	}

	return deleted, err
}

func (p *Persistence) CreateEntry(entry models.Entry) (*models.Entry, *models.ErrorResponse) {
	ent, err := p.Api.CreateEntry(entry)
	if err == nil {
		p.entry.linkAttribute(ent)
		p.entry.addAndSort(ent)

		// Notify for updates
		p.Update.notifyForUpdates(models.NewUpdateWithData([]int{}, []*models.Entry{}, []*models.Entry{ent}))
	}

	return ent, err
}

func (p *Persistence) CreateEntries(entries []*models.Entry) ([]*models.Entry, *models.BulkResponse[models.Entry], *models.ErrorResponse) {
	ent, resp, err := p.Api.CreateEntries(entries)
	if err == nil && len(ent) > 0 {
		p.entry.linkAttributes(&ent)
		p.entry.addAndSort(ent...)

		// Notify for updates
		p.Update.notifyForUpdates(models.NewUpdateWithData([]int{}, []*models.Entry{}, ent))
	}

	return ent, resp, err
}

func (p *Persistence) UpdateEntry(entry *models.Entry) (*models.Entry, *models.ErrorResponse) {
	newEnt, err := p.Api.UpdateEntry(entry)
	if err == nil {
		p.entry.mux.Lock()

		// Remove the netry first
		for i, e := range p.entry.data {
			if entry.ID == e.ID {
				utils.Remove(&p.entry.data, i)
				break
			}
		}

		// Add it again sorted
		p.entry.addAndSortWithoutLock(newEnt)

		p.entry.mux.Unlock()

		// Notify for updates
		p.Update.notifyForUpdates(models.NewUpdateWithData([]int{}, []*models.Entry{newEnt}, []*models.Entry{}))
	}

	return newEnt, err
}

func (p *Persistence) UpdateEntries(entries []*models.Entry) ([]*models.Entry, *models.BulkResponse[models.Entry], *models.ErrorResponse) {
	updated, resp, err := p.Api.UpdateEntries(entries)
	if err == nil && len(updated) > 0 {
		entCopied := updated
		p.entry.mux.Lock()

		// Remove the entries first
		utils.Filter(&entCopied, &p.entry.data, func(a *models.Entry, b *models.Entry) bool { return a.ID == b.ID })
		// And add them sorted afterwards again
		p.entry.linkAttributes(&updated)
		p.entry.addAndSortWithoutLock(updated...)

		p.entry.mux.Unlock()

		// Notify for updates
		p.Update.notifyForUpdates(models.NewUpdateWithData([]int{}, updated, []*models.Entry{}))
	}

	return updated, resp, err
}

func (p *Persistence) PatchEntries(entries []*models.Entry) ([]*models.Entry, *models.BulkResponse[models.Entry], *models.ErrorResponse) {
	updated, resp, err := p.Api.PatchEntries(entries)
	if err == nil && len(updated) > 0 {
		entCopied := updated
		p.entry.mux.Lock()

		// Remove the entries first
		utils.Filter(&entCopied, &p.entry.data, func(a *models.Entry, b *models.Entry) bool { return a.ID == b.ID })
		// And add them sorted afterwards again
		p.entry.linkAttributes(&updated)
		p.entry.addAndSortWithoutLock(updated...)

		p.entry.mux.Unlock()

		// Notify for updates
		p.Update.notifyForUpdates(models.NewUpdateWithData([]int{}, updated, []*models.Entry{}))
	}

	return updated, resp, err
}

// handleUpdate handles the merge of the given update for the locally
// cached data
func (p *persistenceEntry) handleUpdate(upd models.UpdateData[*models.Entry]) {
	p.mux.Lock()

	// Remove deleted entries
	if len(upd.Deleted) > 0 {
		utils.Filter(&p.data, &upd.Deleted, func(a *models.Entry, b int) bool { return a.ID == b })
	}

	// Add created entries
	if len(upd.Created) > 0 {
		p.linkAttributes(&upd.Created)
		p.addAndSortWithoutLock(upd.Created...)
	}

	// Update updated entries
	if len(upd.Updated) > 0 {
		entCopied := upd.Updated
		// Remove the entries first
		utils.Filter(&entCopied, &p.data, func(a *models.Entry, b *models.Entry) bool { return a.ID == b.ID })
		// And add them sorted afterwards again
		p.linkAttributes(&upd.Updated)
		p.addAndSortWithoutLock(upd.Updated...)
	}

	p.mux.Unlock()
}
