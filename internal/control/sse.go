package control

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Raxuis/chaosproxy/internal/events"
)

func (handler *Handler) streamEvents(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("X-Accel-Buffering", "no")

	subscription := handler.bus.Subscribe()
	defer subscription.Close()
	for _, event := range subscription.History() {
		if err := writeEvent(writer, event); err != nil {
			return
		}
	}
	if _, err := io.WriteString(writer, ": connected\n\n"); err != nil {
		return
	}
	if err := http.NewResponseController(writer).Flush(); err != nil {
		return
	}

	heartbeats := time.NewTicker(handler.heartbeat)
	defer heartbeats.Stop()
	for {
		select {
		case <-request.Context().Done():
			return
		case <-handler.stopping.Done():
			return
		case event, open := <-subscription.Events():
			if !open || writeEvent(writer, event) != nil {
				return
			}
			if err := http.NewResponseController(writer).Flush(); err != nil {
				return
			}
		case <-heartbeats.C:
			if _, err := io.WriteString(writer, ": heartbeat\n\n"); err != nil {
				return
			}
			if err := http.NewResponseController(writer).Flush(); err != nil {
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
