package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/RPJoshL/RPdb/v4/go/models"
)

// UpdateRequest contains all fields that can be used to query the current
// version of the data
type UpdateRequest struct {

	// The latest verstion number the client is aware of. Only updates that occured later
	// than this version number are returned
	LatestVersion int

	// In addition to the "LatestVersion" all the updates have also been made
	// later than this time
	LaterThan time.Time

	// The maximum version number until which the updates should be returned
	// - version > latestVersion && version <= maxVersion
	MaxVersion int

	// Only the current version of the data is returned instead of
	// the whole data that was changed
	OnlyVersion bool
}

func (api *Api) GetUpdate(updReq UpdateRequest) (*models.Update, *models.ErrorResponse) {
	req := api.GetRequest(fmt.Sprintf("/update/%d", updReq.LatestVersion), "GET", nil)

	// Build URL with all the query parameters
	q := req.URL.Query()
	q.Add("only_version", strconv.FormatBool(updReq.OnlyVersion))
	if !updReq.LaterThan.IsZero() {
		q.Add("later_than", updReq.LaterThan.Format(models.TimeFormat))
	}
	if updReq.MaxVersion != 0 {
		q.Add("max_version", fmt.Sprintf("%d", updReq.MaxVersion))
	}
	req.URL.RawQuery = q.Encode()

	// Execute request
	res, err := api.DoRequest(req, api.GetDefaultClient())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return models.NewUpdate(res.Body), nil
}
