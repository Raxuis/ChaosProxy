package control

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Raxuis/chaosproxy/internal/events"
)

func (h *Handler) streamEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	subscription := h.bus.Subscribe()
	defer subscription.Close()
	for _, event := range subscription.History() {
		if err := writeEvent(w, event); err != nil {
			return
		}
	}
	if _, err := io.WriteString(w, ": connected\n\n"); err != nil {
		return
	}
	if err := http.NewResponseController(w).Flush(); err != nil {
		return
	}

	heartbeats := time.NewTicker(h.heartbeat)
	defer heartbeats.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-h.stopping.Done():
			return
		case event, open := <-subscription.Events():
			if !open || writeEvent(w, event) != nil {
				return
			}
			if err := http.NewResponseController(w).Flush(); err != nil {
				return
			}
		case <-heartbeats.C:
			if _, err := io.WriteString(w, ": heartbeat\n\n"); err != nil {
				return
			}
			if err := http.NewResponseController(w).Flush(); err != nil {
				return
			}
		}
	}
}

func writeEvent(writer io.Writer, event events.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode event: %w", err)
	}
	_, err = fmt.Fprintf(writer, "id: %d\nevent: request\ndata: %s\n\n", event.ID, payload)
	return err
}
