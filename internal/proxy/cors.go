package proxy

import (
	"net/http"
	"strings"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func handlePreflight(writer http.ResponseWriter, request *http.Request, mode config.CORSMode) bool {
	requestedMethod := request.Header.Get("Access-Control-Request-Method")
	if mode != config.CORSReflect || request.Method != http.MethodOptions || requestedMethod == "" {
		return false
	}

	applyReflectCORS(writer.Header(), request)
	writer.Header().Set("Access-Control-Allow-Methods", requestedMethod)
	if requestedHeaders := request.Header.Get("Access-Control-Request-Headers"); requestedHeaders != "" {
		writer.Header().Set("Access-Control-Allow-Headers", requestedHeaders)
	}
	addVary(writer.Header(), "Access-Control-Request-Method")
	addVary(writer.Header(), "Access-Control-Request-Headers")
	writer.WriteHeader(http.StatusNoContent)
	return true
}

func applyResponseCORS(headers http.Header, request *http.Request, mode config.CORSMode) {
	switch mode {
	case config.CORSReflect:
		applyReflectCORS(headers, request)
	case config.CORSOff:
		for key := range headers {
			if strings.HasPrefix(strings.ToLower(key), "access-control-") {
				headers.Del(key)
			}
		}
	}
}

func applyReflectCORS(headers http.Header, request *http.Request) {
	if request == nil {
		return
	}
	origin := request.Header.Get("Origin")
	if origin == "" {
		return
	}

	headers.Set("Access-Control-Allow-Origin", origin)
	headers.Set("Access-Control-Allow-Credentials", "true")
	addVary(headers, "Origin")
}

func addVary(headers http.Header, value string) {
	for _, header := range headers.Values("Vary") {
		for _, existing := range strings.Split(header, ",") {
			if strings.EqualFold(strings.TrimSpace(existing), value) {
				return
			}
		}
	}
	headers.Add("Vary", value)
}
