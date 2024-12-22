package persistence

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RPJoshL/RPdb/v4/go/models"
	"git.rpjosh.de/RPJosh/go-logger"
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lesismal/nbio/nbhttp/websocket"
)

// WebSocket is used to obtain updates of attributes and entries
// in real time.
// For some specific attribute flags like "noDB" or "executeResponse"
// an opened WebSocket connection is also required.
//
// By default, no WebSocket is used.
// Note that the WebSocket should normally not be used without the persistence
// layer. Fields with "Managed by persistence" should not be touched by you
type WebSocket struct {
	// If a WebSocket connection should be used by this client
	UseWebsocket bool

	// The base URL on which the server is listening for WebSocket connections.
	// Defaulting to "wss://rpdb.rpjosh.de/api/socket"
	SocketURL string

	// Managed by persistence: API key used to authenticate against
	// the server
	ApiKey string

	// Manged by persistence: callback function called when receiving a socket message
	OnMessage func(message models.WebSocketMessage)

	// Managed by persistence: base context to use for the WebSocket
	BaseContext context.Context

	// Managed by persistence: the last update to send during handshake
	Update *PersistenceUpdate

	// The currently used websocket connection
	connection *websocket.Conn

	// The context of a single WebSocket connection
	context       context.Context
	cancelContext context.CancelFunc

	// Mutex to synchronize cancel function and connection access
	mtx sync.Mutex

	// This flag provides a toogle to the CloseListener if the WebSocket was closed
	// intentionally from the client or hardly by the server
	wasIntentionallyClosed atomic.Bool

	// Number of failed reconnect attempts
	reconnectAttempts atomic.Int32

	// Ping pong manager for the connection
	pingPong *ClientMgr
}

// webSocketClientMessage is a wrapper around messages that can be sent
// from the client to the WebSocket
type webSocketClientMessage struct {
	ExecutionResponse models.ExecutionResponse `json:"exec_response"`
}

// Start starts a WebSocket connection to the server if "UseWebSocket" is set to true
func (w *WebSocket) Start() {
	if !w.UseWebsocket {
		// WebSocket should not be started
		return
	}
	if w.BaseContext.Err() != nil {
		// Base Context was canceled
		logger.Debug("Not starting WebSocket: base context already canceled")
		return
	}

	// Try to close any old connections
	if err := w.CloseWithMessage(uint16(1000), "Disconnect"); err != nil {
		logger.Warning(err.Error())
	}

	// Increment the reconnect counter
	w.reconnectAttempts.Store(w.reconnectAttempts.Load() + 1)

	// Lock this for all further operations
	w.mtx.Lock()
	defer w.mtx.Unlock()

	// Close any old context
	if w.cancelContext != nil {
		w.cancelContext()
	}
	// Create new context to use
	w.context, w.cancelContext = context.WithCancel(w.BaseContext)

	// Initialize ping pong handler
	w.pingPong = NewClientMgr(KeepaliveTimeout, w.context)
	go w.pingPong.Run()

	// Reset some values
	w.wasIntentionallyClosed.Store(false)

	// Set default logger to use
	logging.DefaultLogger = newNbioLogger()

	// Start engine and dialer
	engine := nbhttp.NewEngine(nbhttp.Config{Context: w.context})
	if err := engine.Start(); err != nil {
		logger.Error("Failed to start nbio engine: %s", err)
	}
	dialer := websocket.Dialer{
		Engine:      engine,
		Upgrader:    w.newUpgrader(),
		DialTimeout: time.Second * 5,
	}

	// Build request with authentication header
	var headers http.Header = make(http.Header, 3)
	headers.Add("Client-Date", time.Now().Format(models.TimeFormat))
	headers.Add("Client-Version", models.LibraryVersion)
	headers.Add("X-Api-Key", w.ApiKey)
	w.Update.versionLock.RLocker().Lock()
	headers.Add("Version", fmt.Sprintf("%d", w.Update.Version))
	headers.Add("Version-Date", w.Update.VersionDate.Format(models.TimeFormat))
	w.Update.versionLock.RLocker().Unlock()

	// Open connection
	con, _, err := dialer.Dial(w.SocketURL, headers)
	if err != nil {
		logger.Warning("Failed to connect to WebSocket: %s", err)
		w.scheduleReconnect()
		return
	}
	w.connection = con

	// Add ping pong handler for keepalive checks
	con.SetReadDeadline(time.Now().Add(KeepaliveTimeout))
	w.pingPong.Add(con)
}

// newUpgrader creates a new websocket.Upgrader which is used to handle
// messages and the close events
func (w *WebSocket) newUpgrader() *websocket.Upgrader {
	u := websocket.NewUpgrader()

	// Ping pong messages are not automatically be send... So this has not the expected behaviour!
	u.KeepaliveTime = KeepaliveTimeout

	u.SetCloseHandler(func(c *websocket.Conn, i int, s string) {
		if w.wasIntentionallyClosed.Load() {
			logger.Debug("Closed WebSocket intentionally from client side")
		} else {
			w.onClose(c, i, s)
		}
	})

	u.OnMessage(func(c *websocket.Conn, messageType websocket.MessageType, data []byte) {
		c.SetDeadline(time.Now().Add(KeepaliveTimeout))
		w.reconnectAttempts.Store(0)
		logger.Trace("Received message from WebSocket: %s", data)

		// Try to convert the received message to an WebSocket message
		var msg models.WebSocketMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			logger.Debug("Received message from WebSocket: %s", data)
			logger.Warning("Failed to unmarshal WebSocket message: %s", err)
		} else if w.OnMessage != nil {
			logger.Debug("Received message from WebSocket with type %q", msg.Type)
			w.OnMessage(msg)
		} else {
			logger.Debug("Received message from WebSocket but no 'OnMessage()' function provided")
		}
	})

	u.SetPongHandler(func(c *websocket.Conn, s string) {
		c.SetDeadline(time.Now().Add(KeepaliveTimeout))
	})

	u.OnClose(func(c *websocket.Conn, err error) {
		if w.wasIntentionallyClosed.Load() {
			logger.Debug("Closed WebSocket intentionally from client side")
		} else {
			errorMessage := ""
			if err != nil {
				errorMessage = err.Error()
			}

			w.onClose(c, 1006, errorMessage)
		}
	})

	return u
}

// onClose handles the closing event of a WebSocket connection that was not
// intentially closed
func (w *WebSocket) onClose(_ *websocket.Conn, i int, s string) {
	if w.reconnectAttempts.Load() <= 1 {
		logger.Info("Closed WebSocket with status %q (%d)", s, i)
	} else {
		logger.Debug("Closed WebSocket with status %q (%d)", s, i)
	}

	// Cancel the context
	w.mtx.Lock()
	// Clear connection and cancel context
	w.connection = nil
	if w.cancelContext != nil {
		w.cancelContext()
	}

	// Create new context to use
	w.context, w.cancelContext = context.WithCancel(w.BaseContext)

	w.mtx.Unlock()

	// Schedule the next reconnect
	w.scheduleReconnect()
}

// scheduleReconnect schedules a reconnect of the WebSocket after a short waiting time
// to not attach the WebSocket server :)
func (w *WebSocket) scheduleReconnect() {
	waitTime := GetReconnectTimeout(int(w.reconnectAttempts.Load()))
	logger.Debug("Scheduled a reconnect in %.0f seconds", waitTime.Seconds())

	go func() {
		select {
		case <-time.After(waitTime):
			w.Start()
		case <-w.context.Done():
			logger.Debug("Not rescheduling an reconnect because context was canceled")
		}
	}()
}

type Timeout struct {
	max     int
	timeout time.Duration
}

// GetReconnectTimeout returns a timeout for the next reconnect attemt to the websocket
// based on the provided retry count.
// With a higher counter, the wait time will increase
func GetReconnectTimeout(retries int) time.Duration {
	timeouts := []Timeout{
		{2, 5 * time.Second},
		{6, 10 * time.Second},
		{10, 120 * time.Second},
		{15, 5 * time.Minute},
		{25, 10 * time.Minute},
		{50, 30 * time.Minute},
		{90, 60 * time.Minute},
	}

	for _, timeout := range timeouts {
		if retries < timeout.max {
			return timeout.timeout
		}
	}

	return 90 * time.Minute
}

// CloseWithMessage tries to close the WebSocket with the given reason
func (w *WebSocket) CloseWithMessage(code uint16, message string) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	// Check if a connection is available
	if w.context == nil || w.context.Err() != nil || w.connection == nil {
		logger.Trace("Not closing connection because WebSocket is not connected")
		return nil
	}

	// Build reason message
	var codeBytes = make([]byte, 2)
	binary.BigEndian.PutUint16(codeBytes, code)
	codeBytes = append(codeBytes, message...)

	// Set flag that the error handler won't reschedule a reconnect
	w.wasIntentionallyClosed.Store(true)

	// Send the message
	if err := w.connection.WriteMessage(websocket.CloseMessage, codeBytes); err != nil {
		return fmt.Errorf("failed to close the websocket: %s", err)
	} else {
		// Clear connection and cancel context
		w.connection = nil
		if w.cancelContext != nil {
			w.cancelContext()
		}
	}

	return nil
}

// SendExecutionResponse sends the given execution response to the
// WebSocket server
func (w *WebSocket) SendExecutionResponse(response models.ExecutionResponse) {
	data, err := json.Marshal(webSocketClientMessage{ExecutionResponse: response})
	if err != nil {
		logger.Error("Failed to marshal execution response")
		return
	}

	if err := w.sendMessage(data); err != nil {
		logger.Error("Failed to send execution response to WebSocket: %s", err)
	}
}

// sendMessage sends the given message to the current WebSocket
// connection
func (w *WebSocket) sendMessage(data []byte) error {
	if w != nil && w.context.Err() == nil {
		w.mtx.Lock()
		err := w.connection.WriteMessage(websocket.TextMessage, data)
		w.mtx.Unlock()

		return err
	} else {
		return fmt.Errorf("no active connection")
	}
}

// nbioLogger is a logger adapter for the nbio engine to the RPJosh go-logger
type nbioLogger struct {
	*logger.Logger
}

func (l nbioLogger) SetLevel(level int) {
	// Do nothing
}

func (l nbioLogger) Warn(message string, parameters ...any) {
	l.Logger.Log(logger.LevelWarning, message, parameters...)
}

func newNbioLogger() nbioLogger {
	printLevel := logger.GetGlobalLogger().Level
	logLevel := logger.GetGlobalLogger().File.Level

	log := logger.NewLoggerWithFile(
		&logger.Logger{
			Level: printLevel,
			File: &logger.FileLogger{
				Level: logLevel,
			},
			ColoredOutput: logger.GetGlobalLogger().ColoredOutput,
		}, logger.GetGlobalLogger(),
	)

	return nbioLogger{Logger: log}
}
