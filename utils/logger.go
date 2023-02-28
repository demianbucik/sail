package utils

import (
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/apex/log"
)

var DefaultJSONLogHandler = NewJSONLogHandler(os.Stderr)

type JSONLogHandler struct {
	*json.Encoder
	mu sync.Mutex
}

func NewJSONLogHandler(w io.Writer) *JSONLogHandler {
	return &JSONLogHandler{
		Encoder: json.NewEncoder(w),
	}
}

type Entry struct {
	log.Entry
	Level log.Level `json:"severity"` // GCP expects a "severity" field
}

func (h *JSONLogHandler) HandleLog(e *log.Entry) error {
	entry := &Entry{
		Entry: *e,
		Level: e.Level,
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.Encoder.Encode(entry)
}
