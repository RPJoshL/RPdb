package persistence

import (
	"sort"
	"sync"

	"github.com/RPJoshL/RPdb/v4/go/api"
	"github.com/RPJoshL/RPdb/v4/go/models"
	"github.com/RPJoshL/RPdb/v4/go/pkg/utils"
)

type persistenceAttribute struct {
	// API interface to load the data from (Persistence)
	api api.Apiler

	data []*models.Attribute

	// Mutex to synchronize the access to the data
	mux sync.RWMutex
}

func (p *persistenceAttribute) loadData() error {
	attr, err := p.api.GetRealApi().GetAttributes()
	if err != nil {
		return err
	}

	// Update locally stored data by replacing the value
	p.mux.Lock()
	p.data = attr
	p.mux.Unlock()

	return nil
}

// addAndSortWithoutLock adds all the given attributes to the local cache and sorts the whole
// array again.
// This method does NOT lock the data mutex
func (p *persistenceAttribute) addAndSortWithoutLock(attributes ...*models.Attribute) {
	p.data = append(p.data, attributes...)
	sort.SliceStable(p.data, func(i, j int) bool {
		return p.data[i].Name < p.data[j].Name
	})
}

// GetEntriesAll is the same function as "GetAttributes()" without
// any filter condition
func (p *Persistence) GetAttributesAll() []*models.Attribute {
	// No error can occure while fetching the entries locally
	attr, _ := p.GetAttributes()

	return attr
}

func (p *Persistence) GetAttribute(id int) (*models.Attribute, *models.ErrorResponse) {
	p.attribute.mux.RLocker().Lock()
	defer p.attribute.mux.RLocker().Unlock()

	for i := range p.attribute.data {
		if (p.attribute.data[i]).ID == id {
			return p.attribute.data[i], nil
		}
	}

	return nil, &models.ErrorResponse{ID: "ATTRIBUTE_NOT_FOUND", ResponseCode: 404, Message: "Attribute was not found"}
}

func (p *Persistence) GetAttributes() (rtc []*models.Attribute, err *models.ErrorResponse) {
	// Array is coppied during reassignment
	p.attribute.mux.RLocker().Lock()
	rtc = p.attribute.data
	p.attribute.mux.RLocker().Unlock()

	return
}

// GetAttributeByName returns an attribute with the given, unique name.
// If the attribute does not exist an error is returned
func (p *Persistence) GetAttributeByName(name string) (*models.Attribute, *models.ErrorResponse) {
	p.attribute.mux.RLock()
	defer p.attribute.mux.RUnlock()

	for i := range p.attribute.data {
		if p.attribute.data[i].Name == name {
			return p.attribute.data[i], nil
		}
	}

	return nil, &models.ErrorResponse{ID: "ATTRIBUTE_NOT_FOUND", ResponseCode: 404, Message: "Attribute was not found"}
}

// handleUpdate handles the merge of the given update for the locally
// cached data
func (p *persistenceAttribute) handleUpdate(upd models.UpdateData[*models.Attribute]) {
	p.mux.Lock()

	// Remove deleted entries
	if len(upd.Deleted) > 0 {
		utils.Filter(&upd.Deleted, &p.data, func(a int, b *models.Attribute) bool { return a == b.ID })
	}

	// Add created entries
	if len(upd.Created) > 0 {
		p.addAndSortWithoutLock(upd.Created...)
	}

	// Update updated entries
	if len(upd.Updated) > 0 {
		attrCopy := upd.Updated
		// Remove the entries first
		utils.Filter(&attrCopy, &p.data, func(a *models.Attribute, b *models.Attribute) bool { return a.ID == b.ID })
		// And add them sorted afterwards again
		p.addAndSortWithoutLock(upd.Updated...)
	}

	p.mux.Unlock()
}
