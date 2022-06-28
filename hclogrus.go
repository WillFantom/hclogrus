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
	checkID     string
	failLevels  []logrus.Level
	messageSent chan *LogMessage
}

type LogMessage struct {
	Level       int            `json:"level"`
	LevelString string         `json:"level_string"`
	Time        time.Time      `json:"time"`
	Message     string         `json:"message"`
	Data        map[string]any `json:"data"`
	Ticker      bool           `json:"ticker"`
}

var (
	baseURL       string      = "https://hc-ping.com"
	tickerMessage *LogMessage = &LogMessage{
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
// returned if an initial check ping fails.
func New(checkID string, tickDuration time.Duration, failLevels ...logrus.Level) (*HCLogrusHook, error) {
	h := &HCLogrusHook{
		checkID:     checkID,
		failLevels:  failLevels,
		messageSent: make(chan *LogMessage),
	}
	go h.tick(tickDuration)
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
// returned...
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
	h.messageSent <- message
	return nil
}

func (h *HCLogrusHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *HCLogrusHook) sendLogMessage(lm *LogMessage) error {
	m, _ := json.Marshal(lm)
	messageBytes := bytes.NewBuffer(m)
	u, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("failed to create healthchecks.io ping url: %w", err)
	}
	u.Path = path.Join(u.Path, h.checkID)
	for _, l := range h.failLevels {
		if logrus.Level(lm.Level) == l {
			u.Path = path.Join(u.Path, "fail")
		}
	}
	request, err := http.NewRequest("POST", u.String(), messageBytes)
	if err != nil {
		return fmt.Errorf("failed to create log message for healthchecks.io")
	}
	resp, err := new(http.Client).Do(request)
	if err != nil || resp.StatusCode != 200 {
		return fmt.Errorf("failed to send log entry to healthchecks.io: %w", err)
	}
	return nil
}

func (h *HCLogrusHook) tick(period time.Duration) {
	message := tickerMessage
	for {
		message.Ticker = true
		select {
		case <-time.After(period):
			h.sendLogMessage(message)
		case m, ok := <-h.messageSent:
			if !ok {
				return
			}
			message = m
		}
	}
}
