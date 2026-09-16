package control

import (
	"errors"
	"net/http"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func (h *Handler) getScenarios(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.runtime.Scenarios())
}

func (h *Handler) resetScenario(w http.ResponseWriter, r *http.Request) {
	if err := h.runtime.ResetScenario(r.PathValue("name")); err != nil {
		if errors.Is(err, config.ErrScenarioNotFound) {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
