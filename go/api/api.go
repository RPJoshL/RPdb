// api takes care of executing request to the server and returning the
// response.
package api

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/RPJoshL/RPdb/v4/go/models"
	"github.com/RPJoshL/RPdb/v4/go/pkg/language"
	"git.rpjosh.de/RPJosh/go-logger"
)

// Api contains the shared ressources needed for the client requests.
// You should initialize this struct only over the "New()" function.
// This struct implements "Apiler"
type Api struct {
	// API Key of the client
	apiKey string

	// Context of every request
	ctx context.Context

	ApiOptions
}

// ApiOptions specifies some additional options for the client.
// These are completely optional and can all be empty
type ApiOptions struct {

	// If this client should be treated like the Java clients.
	// That does mean that the server thinks that shared ressources like
	// the attribute should not be expanded fully, because the client does
	// already have all available attributes cached locally
	TreatAsJavaClient bool

	// Force the use of as specific language. This is a two-digit code (ISO 639).
	// By default, we try to get the os language or use English as a default language
	Language string

	// When running multiple instances with the same API-Key (WHICH IS NOT RECOMMENDED)
	// you should set this flag to true that this client is also notified when an entry or
	// attribute were changed
	MultiInstance bool

	// Endpoint of the api to send all requests to.
	// Defaulting to https://rpdb.rpjosh.de/api/v1
	BaseUrl string
}

// Apiler contains all methods for making requests against the API
type Apiler interface {
	GetEntry(id int) (*models.Entry, *models.ErrorResponse)
	GetEntries(filter models.EntryFilter) ([]*models.Entry, *models.ErrorResponse)
	CreateEntry(entry models.Entry) (*models.Entry, *models.ErrorResponse)
	DeleteEntry(id int) (*models.ResponseMessageWrapper, *models.ErrorResponse)
	UpdateEntry(entry *models.Entry) (*models.Entry, *models.ErrorResponse)
	CreateEntries(entries []*models.Entry) ([]*models.Entry, *models.BulkResponse[models.Entry], *models.ErrorResponse)
	UpdateEntries(entries []*models.Entry) ([]*models.Entry, *models.BulkResponse[models.Entry], *models.ErrorResponse)
	PatchEntries(entries []*models.Entry) ([]*models.Entry, *models.BulkResponse[models.Entry], *models.ErrorResponse)
	DeleteEntries(idsToDelete []int) ([]int, *models.BulkResponse[int], *models.ErrorResponse)
	DeleteEntriesFiltered(filter models.EntryFilter) (EntryDeleteFiltered, *models.ErrorResponse)

	// MarkEntryAsExecuted marks the entry with the given ID as executed. This does
	// only work for attributes with the flag EA
	MarkEntryAsExecuted(id int) *models.ErrorResponse

	GetUpdate(updReq UpdateRequest) (*models.Update, *models.ErrorResponse)

	GetAttribute(id int) (*models.Attribute, *models.ErrorResponse)
	GetAttributeByName(name string) (*models.Attribute, *models.ErrorResponse)
	GetAttributes() ([]*models.Attribute, *models.ErrorResponse)

	// GetRealApi should always return the underlaying API that directly executes the api requests
	// without any persistence layer
	GetRealApi() Apiler
}

func (a *Api) GetRealApi() Apiler {
	return a
}

// setAndValidateDefaults sets some default values if no value was given
// and validates the given options (very basic)
func (options *ApiOptions) setAndValidateDefaults() {
	if options.Language == "" {
		options.Language = language.GetOsLanguage("en")
	}

	if options.BaseUrl == "" {
		options.BaseUrl = "https://rpdb.rpjosh.de/api/v1"
	} else {
		if strings.HasSuffix("/", options.BaseUrl) {
			options.BaseUrl = strings.TrimRight("/", options.BaseUrl)
		}
	}
}

// NewApi is a wrapper for "NewApiWithContext" using context.Background.
// You have to provide the api key and optional options.
func NewApi(apiKey string, options ApiOptions) *Api {
	return NewApiWithContext(context.Background(), apiKey, options)
}

// NewApiWithContext initializes a new api client.
// You have to provide the api key and optional options.
func NewApiWithContext(context context.Context, apiKey string, options ApiOptions) *Api {

	// Set some default values
	options.setAndValidateDefaults()

	return &Api{
		apiKey:     apiKey,
		ctx:        context,
		ApiOptions: options,
	}
}

// GetRequest returns an authenticated http request and the
// required headers based on the previously given api options.
// The given path should be relative to the base url: '/entry/123'.
// The body can be nil
func (api *Api) GetRequest(path string, method string, body io.Reader) *http.Request {
	logger.Trace("Executing request: %s %s", method, path)
	req, err := http.NewRequestWithContext(api.ctx, method, api.BaseUrl+path, body)
	if err != nil {
		logger.Error("Failed to create request: %s", err)
		return nil
	}

	// Set required headers
	req.Header.Set("X-Api-Key", api.apiKey)
	req.Header.Set("Java-Client", strconv.FormatBool(api.TreatAsJavaClient))
	req.Header.Set("Language", api.Language)
	req.Header.Set("Multi-Instance", strconv.FormatBool(api.MultiInstance))
	req.Header.Set("Client-Version", models.LibraryVersion)

	// When fetching entries the dateTime has to be adjusted for the client
	// time zone. So the current client date will be sent
	if strings.HasPrefix(path, "/entry") {
		req.Header.Set("Client-Date", time.Now().Format(models.TimeFormat))
	}

	// The body is always sent in JSON format
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	return req
}

// GetDefaultClient returns a new http.Client with default
// settings
func (api *Api) GetDefaultClient() http.Client {
	return http.Client{Timeout: 10 * time.Second}
}

// ExecuteRequests executes the given request and pretifies occured errors.
// See "GetRequest()" for more information.
// This does internally use a new http.client every time. If you are making a huge number
// of requests you should consider reusing the same client for not open a connection every time!
func (api *Api) ExecuteRequest(path string, method string, body io.Reader) (*http.Response, *models.ErrorResponse) {
	client := api.GetDefaultClient()
	request := api.GetRequest(path, method, body)

	return api.DoRequest(request, client)
}

// execute executes the response and returns the result.
// Status codes >= 500 are handled as errors and will be returned
// as an ErrorResponse.
func (api *Api) execute(request *http.Request, client http.Client) (path string, response *http.Response, error *models.ErrorResponse) {
	response, err := client.Do(request)

	path = request.Method + ` "` + strings.Replace(request.URL.String(), api.BaseUrl, "", 1) + `"`
	if err != nil {
		// An error occured
		return path, nil, &models.ErrorResponse{ErrorGo: err, Path: path}
	}

	// Unknown server error
	if response.StatusCode >= 500 {
		body, errRead := ioutil.ReadAll(response.Body)
		if errRead != nil {
			logger.Error("An unknown error occured while queuing the server: %s", path)
			return path, nil, &models.ErrorResponse{ErrorGo: err, Path: path, ResponseCode: response.StatusCode}
		}
		logger.Error("An unknown error occured while queuing the server: %s\nBody: %s", path, body)
		return path, nil, &models.ErrorResponse{Message: "Unknown error", Path: path, ResponseCode: response.StatusCode}
	}

	// Don't process the response furthermore (body can only be read once)
	return
}

// handlePHPError reads the error from a request that failed with a
// status code between 300 - 499.
// In almost all cases this should be a sepcific error that the PHP server
// throwed to the client controlled
func (api *Api) handlePHPError(body []byte, res *http.Response, path string, request *http.Request) *models.ErrorResponse {
	var errorResponse struct {
		R models.ErrorResponse `json:"error"`
	}
	errorResponse.R.Path = path
	errorResponse.R.ResponseCode = res.StatusCode
	if json.Unmarshal(body, &errorResponse) == nil && errorResponse.R.Message != "" {
		// It was a valid error response
		logger.Debug(errorResponse.R.PrintLog("  "))
		return &errorResponse.R
	} else {
		// Error from webserver?
		logger.Error("An unknown error occured while queuing the server: %s\nBody: %s", path, body)
		return &models.ErrorResponse{Message: "Unknown error", Path: path, ResponseCode: res.StatusCode}
	}
}

// DoRequest executes the given request with the client.
// Occurred errors are checked and proceeded and will be returned
// wrapped as a custom error.
//
// Note: for bulk responses you should use the public function "DoRequestBulk()"
func (api *Api) DoRequest(request *http.Request, client http.Client) (*http.Response, *models.ErrorResponse) {

	// Execute the request
	path, res, err := api.execute(request, client)
	if err != nil {
		return res, err
	}

	// Invalid request send to the server (status code 3xx and 4xx).
	// A custom error is returned in such a case
	if res.StatusCode >= 300 {

		// Read the body of the request
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			logger.Debug("Failed to read response body: %s", err)
			logger.Error("An unknown error occured while queuing the server: %s %q (%d)", request.Method, request.URL, res.StatusCode)
			return nil, &models.ErrorResponse{ErrorGo: err, Path: path, ResponseCode: res.StatusCode}
		}

		return nil, api.handlePHPError(body, res, path, request)
	}

	return res, nil
}

// DoRequestBulk executes the given bulk request with the api client.
// This method should only be used for BULK endpoints at all.
//
// The handling of errors > 500 is the same as in "api.DoRequest()".
// For errors < 500 a bulk response is returned. Look at the errors
// of the bulk response itself for error handling.
//
// While calling this function you have to provide the generic type the
// bulk response should lead to. This is for example an entry or an Integer
func DoRequestBulk[T any](api *Api, request *http.Request, client http.Client) (*models.BulkResponse[T], *models.ErrorResponse) {

	// Execute the request
	path, res, err := api.execute(request, client)
	if err != nil {
		return nil, err
	}

	// Read the body once
	body, ioErr := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Debug("Failed to read response body: %s", ioErr)
		logger.Error("An unknown error occured while queuing the server: %s %q (%d)", request.Method, request.URL, res.StatusCode)
		return nil, &models.ErrorResponse{ErrorGo: err, Path: path, ResponseCode: res.StatusCode}
	}
	defer res.Body.Close()

	// If a status code >= 300 is returned there are two possibilities:
	//  - The request was correct, but individual operations failed
	//  - The request was not correct (authentication, params, ...)
	// So we first try to convert the body to an ErrorResponse.
	var rtc models.BulkResponse[T]
	if err := json.Unmarshal(body, &rtc); err != nil {
		logger.Debug("Did probably not receive a bulk response: %s", err)

		// No bulk response received ⇾ it should be a "normal" error
		return nil, api.handlePHPError(body, res, path, request)
	}

	// Log the bulk request on errors
	if !rtc.WasSuccessful() {
		logger.Debug("Bulk request for %s failed:\n%s", path, rtc)
	}

	// Received a bulk response
	return &rtc, nil
}
