package api

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/RPJoshL/RPdb/v4/go/models"
	"git.rpjosh.de/RPJosh/go-logger"
)

type bulkEntry[T any] struct {
	Data []T `json:"bulk"`
}

// EntryDeleteFiltered is used to return the deleted entries when they were
// deleted based on filter values
type EntryDeleteFiltered struct {
	// How many entries were deleted
	Count int `json:"count"`

	// The IDs of the deleted entries
	IDs []int `json:"ids"`

	Message models.ResponseMessage `json:"message"`
}

func (e *bulkEntry[T]) toJson() []byte {
	rtc, err := json.Marshal(e)
	if err != nil {
		logger.Warning("Failed to marshal bulk entries: %s", err)
		return []byte("{}")
	} else {
		return rtc
	}
}

func (api *Api) GetEntry(id int) (*models.Entry, *models.ErrorResponse) {
	res, err := api.ExecuteRequest(fmt.Sprintf("/entry/%d", id), "GET", nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return models.NewEntry(res.Body), nil
}

func (api *Api) GetEntries(filter models.EntryFilter) ([]*models.Entry, *models.ErrorResponse) {
	res, err := api.ExecuteRequest("/entry", "PROPFIND", bytes.NewBuffer(filter.ToJson()))
	if err != nil {
		return []*models.Entry{}, err
	}

	// No entries received
	if res.StatusCode == 204 {
		return []*models.Entry{}, nil
	}

	defer res.Body.Close()
	var rtc []*models.Entry
	if err := json.NewDecoder(res.Body).Decode(&rtc); err != nil {
		logger.Debug("Failed to decode entry array: %s", err)
		return []*models.Entry{}, &models.ErrorResponse{ErrorGo: err}
	}

	return rtc, nil
}

func (api *Api) CreateEntry(entry models.Entry) (*models.Entry, *models.ErrorResponse) {
	res, err := api.ExecuteRequest("/entry", "POST", bytes.NewBuffer(entry.ToJson()))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return models.NewEntry(res.Body), nil
}

func (api *Api) DeleteEntry(id int) (*models.ResponseMessageWrapper, *models.ErrorResponse) {
	res, err := api.ExecuteRequest(fmt.Sprintf("/entry/%d", id), "DELETE", nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return models.NewResponseMessageWrapper(res.Body), nil
}

func (api *Api) UpdateEntry(entry *models.Entry) (*models.Entry, *models.ErrorResponse) {
	res, err := api.ExecuteRequest(fmt.Sprintf("/entry/%d", entry.ID), "PUT", bytes.NewBuffer(entry.ToJson()))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return models.NewEntry(res.Body), nil
}

func (api *Api) CreateEntries(entries []*models.Entry) ([]*models.Entry, *models.BulkResponse[models.Entry], *models.ErrorResponse) {
	return api.makeBulkCreateOrUpdate("POST", entries)
}

func (api *Api) UpdateEntries(entries []*models.Entry) ([]*models.Entry, *models.BulkResponse[models.Entry], *models.ErrorResponse) {
	return api.makeBulkCreateOrUpdate("PUT", entries)
}

func (api *Api) PatchEntries(entries []*models.Entry) ([]*models.Entry, *models.BulkResponse[models.Entry], *models.ErrorResponse) {
	return api.makeBulkCreateOrUpdate("PATCH", entries)
}

func (api *Api) makeBulkCreateOrUpdate(method string, entries []*models.Entry) ([]*models.Entry, *models.BulkResponse[models.Entry], *models.ErrorResponse) {
	// Get request with data
	ent := bulkEntry[*models.Entry]{Data: entries}
	req := api.GetRequest("/entry", method, bytes.NewBuffer(ent.toJson()))

	// Execute request
	resp, err := DoRequestBulk[models.Entry](api, req, api.GetDefaultClient())
	if err != nil {
		return nil, nil, err
	}

	// Get created entries
	rtc := make([]*models.Entry, 0)
	for i, e := range resp.ResponseData {
		if e.Status == models.StatusCreated || e.Status == models.StatusUpdated {
			rtc = append(rtc, &resp.ResponseData[i].Data)
		}
	}

	return rtc, resp, nil
}

func (api *Api) DeleteEntries(idsToDelete []int) ([]int, *models.BulkResponse[int], *models.ErrorResponse) {
	// Get request with data
	ent := bulkEntry[int]{Data: idsToDelete}
	req := api.GetRequest("/entry/delete", "PATCH", bytes.NewBuffer(ent.toJson()))

	// Execute request
	resp, err := DoRequestBulk[int](api, req, api.GetDefaultClient())
	if err != nil {
		return nil, nil, err
	}

	// Get deleted entries
	rtc := make([]int, 0)
	for _, e := range resp.ResponseData {
		if e.Status == models.StatusDeleted {
			rtc = append(rtc, e.Data)
		}
	}

	return rtc, resp, nil
}

func (api *Api) DeleteEntriesFiltered(filter models.EntryFilter) (EntryDeleteFiltered, *models.ErrorResponse) {
	res, err := api.ExecuteRequest("/entry/delete", "PATCH", bytes.NewBuffer(filter.ToJson()))
	if err != nil {
		return EntryDeleteFiltered{}, err
	}

	defer res.Body.Close()
	var rtc EntryDeleteFiltered
	if err := json.NewDecoder(res.Body).Decode(&rtc); err != nil {
		logger.Debug("Failed to decode delete entry filtered response: %s", err)
		return EntryDeleteFiltered{}, &models.ErrorResponse{ErrorGo: err}
	}

	return rtc, nil
}

func (api *Api) MarkEntryAsExecuted(id int) *models.ErrorResponse {
	res, err := api.ExecuteRequest(fmt.Sprintf("/api-key/execution/%d", id), "POST", nil)
	if err == nil {
		res.Body.Close()
	}
	return err
}
