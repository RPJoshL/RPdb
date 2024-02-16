package api

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/RPJoshL/RPdb/v4/go/models"
	"git.rpjosh.de/RPJosh/go-logger"
)

func (api *Api) GetAttribute(id int) (*models.Attribute, *models.ErrorResponse) {
	res, err := api.ExecuteRequest(fmt.Sprintf("/attribute/%d", id), "GET", nil)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	return models.NewAttribute(res.Body), nil
}

func (api *Api) GetAttributeByName(name string) (*models.Attribute, *models.ErrorResponse) {

	// Build query parameters
	params := url.Values{}
	params.Add("name", name)

	res, err := api.ExecuteRequest("/attribute?"+params.Encode(), "GET", nil)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	var rtc []*models.Attribute
	if err := json.NewDecoder(res.Body).Decode(&rtc); err != nil {
		logger.Debug("Failed to decode attribute array: %s", err)
		return nil, &models.ErrorResponse{ErrorGo: err}
	}

	if len(rtc) != 1 {
		return nil, &models.ErrorResponse{ID: "ATTRIBUTE_NOT_FOUND", ResponseCode: 404, Message: "Attribute was not found"}
	}

	return rtc[0], nil
}

func (api *Api) GetAttributes() ([]*models.Attribute, *models.ErrorResponse) {
	res, err := api.ExecuteRequest("/attribute", "GET", nil)
	if err != nil {
		return []*models.Attribute{}, err
	}

	// No attributes received
	if res.StatusCode == 204 {
		return []*models.Attribute{}, nil
	}

	defer res.Body.Close()
	var rtc []*models.Attribute
	if err := json.NewDecoder(res.Body).Decode(&rtc); err != nil {
		logger.Debug("Failed to decode attribute array: %s", err)
		return []*models.Attribute{}, &models.ErrorResponse{ErrorGo: err}
	}

	return rtc, nil
}
