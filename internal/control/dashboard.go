package control

import (
	"net/http"

	"github.com/Raxuis/chaosproxy/web"
)

const dashboardPolicy = "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; " +
	"img-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

func dashboard() http.Handler {
	files := http.FileServerFS(web.Files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", dashboardPolicy)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	})
}

func (h *Handler) getStats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.bus.Stats())
}
