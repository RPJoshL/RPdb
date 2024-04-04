package models

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"git.rpjosh.de/RPJosh/go-logger"
)

// Update is used to apply updates received from the WebSocket (or the API)
// to the locally cached data.
//
// It contains the current version number and all the updates that occurred since the last
// known version of the client
type Update struct {

	// Current version number
	Version int `json:"version"`

	// Date when the last update was triggered
	VersionDate DateTime `json:"version_date"`

	// Updated entries
	Entry UpdateData[*Entry] `json:"entry"`

	// Updated attributes
	Attribute UpdateData[*Attribute] `json:"attribute"`
}

// UpdateData contains the objects that were deleted, updated or created.
type UpdateData[T any] struct {
	// The unique IDs of the deleted objects
	Deleted []int `json:"deleted"`

	// Entrys: deleted objects to trigger the onDelete hook for.
	// You will receive a WebSocket update with these entries even if you
	// deleted them by yourself (because you probably won't have the data anymore :).
	// You can only use the parameter, date of the entry, entry ID and the attribute ID from these data
	DeletedPre []T `json:"deletedPre"`

	// Updated objects with their new data
	Updated []T `json:"updated"`

	// Created objects
	Created []T `json:"created"`
}

func (u Update) String() string {

	// Build entry and attrbute string
	entries := ""
	if u.Entry.IsUpdate() {
		entries += " Entries: " + u.Entry.String()
	}
	attributes := ""
	if u.Attribute.IsUpdate() {
		if u.Entry.IsUpdate() {
			attributes += "  --  "
		}
		attributes += "Attributes: " + u.Attribute.String()
	}

	// Return finished string
	return fmt.Sprintf("[Version %d from %s]%s%s", u.Version, u.VersionDate.FormatPretty(), entries, attributes)
}

// IsZero returns whether this instance is empty.
//
// Note: if you use this as the subject's data for the update observer
// this means that no update information are available because this was
// probably the initial loading ⇾ entries and attributes were "updated"!
func (up *Update) IsZero() bool {
	return up.Version == 0 && up.VersionDate.IsZero()
}

// IsUpdate returns if this specific entity type was updated
func (up *UpdateData[T]) IsUpdate() bool {
	return len(up.Created) != 0 || len(up.Deleted) != 0 || len(up.Updated) != 0
}

func (up UpdateData[T]) String() string {
	return fmt.Sprintf("%d deleted | %d updated | %d created", len(up.Deleted), len(up.Updated), len(up.Created))
}

// NewUpdate decodes the JSON response of the given reader
// to a new Update
func NewUpdate(r io.Reader) *Update {
	var upd Update

	if err := json.NewDecoder(r).Decode(&upd); err != nil {
		logger.Warning("Failed to decode entry: %s", err)
	}

	return &upd
}

// NewUpdateWithData creates a new update object with the current version
// time and the given data
func NewUpdateWithData(deletedEntries []int, updatedEntries []*Entry, createdEntries []*Entry) *Update {
	return &Update{
		VersionDate: DateTime{Time: time.Now()},
		Entry: UpdateData[*Entry]{
			Deleted: deletedEntries,
			Updated: updatedEntries,
			Created: createdEntries,
		},
	}
}
