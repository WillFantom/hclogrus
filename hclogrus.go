package hclogrus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/sirupsen/logrus"
)

// HCLogrusHook allows for log messages to be set to a specific Healthchecks.io
// check.
type HCLogrusHook struct {
	checkURL    string
	failLevels  []logrus.Level
	interval    time.Duration
	messageSent chan LogMessage
}

// LogMessage is the strucutre that log messages will take when sent to
// Healthchecks.io.
type LogMessage struct {
	Level       int            `json:"level"`
	LevelString string         `json:"level_string"`
	Time        time.Time      `json:"time"`
	Message     string         `json:"message"`
	Data        map[string]any `json:"data"`
	Ticker      bool           `json:"ticker"`
}

const (
	JobStartField string = "@hc_job_start"
)

var (
	baseURL       string     = "https://hc-ping.com"
	tickerMessage LogMessage = LogMessage{
		Level:       -1,
		LevelString: "ticker",
		Message:     "",
		Data:        nil,
	}
)

// New creates a new Healthchecks hook to be used with Logrus. The check ID from
// healthchecks.io must be provided, along with the period in which the ticker
// should execte. An optional set of log levels should be provided, that if
// used, will flag the log as an error, marking the check as failed. An error is
// returned if an initial check ping fails or if the ping URL can not be created
// from the provided checkID.
func New(checkID string, tickDuration time.Duration, failLevels ...logrus.Level) (*HCLogrusHook, error) {
	h := &HCLogrusHook{
		checkURL:    "",
		failLevels:  failLevels,
		interval:    tickDuration,
		messageSent: make(chan LogMessage),
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create healthchecks.io ping url: %w", err)
	}
	u.Path = path.Join(u.Path, checkID)
	h.checkURL = u.String()
	go h.tick()
	if err := h.sendLogMessage(&LogMessage{
		Level:       -1,
		LevelString: "startup",
		Time:        time.Now(),
		Message:     "logrus healthchecks hook created",
		Data:        nil,
		Ticker:      true,
	}); err != nil {
		return nil, err
	}
	return h, nil
}

// BaseURL returns the base healthcheck ping URL being used.
func BaseURL() string {
	return baseURL
}

// SetBaseURL sets the base healthcheck ping URL being used.
func SetBaseURL(base string) {
	baseURL = base
}

// Fire is called when logging by Logrus. It creates a LogMessage from a Logrus
// entry and sends this to healthchecks.io. If the hc-ping fails, no error is
// returned... (in order to keep things speedy).
func (h *HCLogrusHook) Fire(entry *logrus.Entry) error {
	message := &LogMessage{
		Level:       int(entry.Level),
		LevelString: entry.Level.String(),
		Time:        entry.Time,
		Message:     entry.Message,
		Data:        entry.Data,
		Ticker:      false,
	}
	go h.sendLogMessage(message)
	h.messageSent <- *message
	return nil
}

func (h *HCLogrusHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// SetTickerInterval allows the interval at which ticker messages are sent to be
// changed.
func (h *HCLogrusHook) SetTickerInterval(interval time.Duration) {
	h.interval = interval
}

func (h *HCLogrusHook) sendLogMessage(lm *LogMessage) error {
	m, _ := json.Marshal(lm)
	messageBytes := bytes.NewBuffer(m)
	endpoint := h.checkURL
	if h.isFailureEntry(logrus.Level(lm.Level)) {
		// TODO: Use Go 1.19 url path join method (when not in beta)
		endpoint = fmt.Sprintf("%s/%s", endpoint, "fail")
	} else if h.isJobStartEntry(lm.Data) && !lm.Ticker {
		// TODO: Use Go 1.19 url path join method (when not in beta)
		endpoint = fmt.Sprintf("%s/%s", endpoint, "start")
	}
	request, err := http.NewRequest("POST", endpoint, messageBytes)
	if err != nil {
		return fmt.Errorf("failed to create log message for healthchecks.io")
	}
	hc := http.Client{
		Timeout: time.Second * 2,
	}
	if _, err := hc.Do(request); err != nil {
		return fmt.Errorf("failed to send log entry to healthchecks.io: %w", err)
	}
	return nil
}

func (h *HCLogrusHook) isFailureEntry(level logrus.Level) bool {
	for _, l := range h.failLevels {
		if logrus.Level(level) == l {
			return true
		}
	}
	return false
}

func (h *HCLogrusHook) isJobStartEntry(entryData logrus.Fields) bool {
	if startField, ok := entryData[JobStartField]; ok {
		if isStart, ok := startField.(bool); ok {
			return isStart
		}
	}
	return false
}

func (h *HCLogrusHook) tick() {
	message := tickerMessage
	for {
		message.Ticker = true
		select {
		case <-time.After(h.interval):
			h.sendLogMessage(&message)
		case m, ok := <-h.messageSent:
			if !ok {
				return
			}
			message = m
		}
	}
}
