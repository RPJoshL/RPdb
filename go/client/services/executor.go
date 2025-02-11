package service

import (
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/RPJoshL/RPdb/v4/go/client/models"
	mod "github.com/RPJoshL/RPdb/v4/go/models"
	"github.com/RPJoshL/RPdb/v4/go/persistence"
	"git.rpjosh.de/RPJosh/go-logger"
)

// ProgramExecutor handles the exeuction of entries. A program with the
// parameter of the entry and additional details like the dateTime, attributeName and
// the entryId is called configured by the given AttributeOptions
type ProgramExecutor struct {

	// A map indexed by the attribute ID with the attribute properties
	Attributes map[int]models.AttributeOptions

	// Mutex to sync the execution
	Mutex *sync.Mutex
}

// Execute calls a program defined in the attribute options
func (e *ProgramExecutor) Execute(ent mod.Entry, typ persistence.ExecutionType) {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	// Get the attribute to execute
	attr, doesExist := e.Attributes[ent.Attribute.ID]
	if !doesExist {
		return
	}
	program := ""
	logMessage := ""
	switch typ {
	case persistence.DEFAULT:
		program = attr.Program
		logMessage = "Executing entry"
	case persistence.DELETE:
		program = attr.OnDeleteProgram
		logMessage = "Executing delete hook for entry"
	default:
		logger.Warning("Received unknown execution type: %q", typ)
	}

	// Nothing to execute
	if program == "" {
		return
	}

	logger.Info("%s %s with attribute %q (#%d)", logMessage, ent.DateTime.FormatPretty(), ent.Attribute.Name, ent.ID)

	// Get the CLI parameters
	params := e.getParameters(&ent, attr)

	// Call the programm and detach its process
	if err := e.startProgramm(program, params); err != nil {
		logger.Warning("Failed to start %q: %s", attr.Program, err)
	}
}

// ExecuteResponse calls a program defined in the attribute options and returns
// the exeuction response.
// Therefore, this method does block until the program was executed
func (e *ProgramExecutor) ExecuteResponse(ent mod.Entry) (rtc *mod.ExecutionResponse) {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	rtc = &mod.ExecutionResponse{
		EntryId: ent.ID,
	}

	// Get the attribute to execute
	attr, doesExist := e.Attributes[ent.Attribute.ID]
	if !doesExist || attr.Program == "" {
		return nil
	}

	// Do not return a response
	if attr.HideResponse {
		logger.Debug("Returning a response is diabled in config")
		e.Execute(ent, persistence.DEFAULT)
		return nil
	}

	logger.Info("Executing entry %s (#%d) and returning response", ent.DateTime.FormatPretty(), ent.ID)

	// Get the CLI parameters
	params := e.getParameters(&ent, attr)

	// Call the program (in foreground) and return response
	cmd := exec.Command(attr.Program, params...)
	// Combine stdout and stderr
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		logger.Warning("%s", err.Error())
	}
	cmd.Stderr = cmd.Stdout
	defer cmdReader.Close()

	// Function to read the combined output
	go func() {
		outCombined, err := io.ReadAll(cmdReader)
		if err != nil {
			logger.Warning("Failed to read output from program %q: %s", attr.Program, err)
		}
		rtc.Text = string(outCombined)
	}()

	// Execute it
	err = cmd.Run()

	// If a non-zero return code was returned, an error is returned in go
	if err != nil {
		if werr, ok := err.(*exec.ExitError); ok {
			rtc.Code = werr.ExitCode()
		} else {
			logger.Warning("Error during execution of program %q: %s", attr.Program, err)
			rtc.Text += err.Error()
			rtc.Code = -1
		}
	}

	// Hide response for return code 124
	if rtc.Code == 124 {
		logger.Debug("Return code is 124. Not returning a response")
		return nil
	}

	return
}

// getParameters returns a list of parameters that should be used to call the program
func (e *ProgramExecutor) getParameters(ent *mod.Entry, attr models.AttributeOptions) []string {
	// Build dynamic parameters
	parameters := make([]string, len(ent.Parameters))
	for i, p := range ent.Parameters {
		parameters[i] = p.GetValue(ent.Attribute)
	}

	// Only call the program with the parameters with entries detail
	if attr.PassOnlyParameter {
		return parameters
	}

	return append(parameters, []string{
		ent.DateTime.Format(mod.TimeFormat),
		ent.Attribute.Name,
		fmt.Sprintf("%d", ent.ID),
	}...)
}
