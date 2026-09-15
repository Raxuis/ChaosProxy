package proxy

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/loopback"
)

type corsPolicy struct {
	mode      config.CORSMode
	anyOrigin bool
	origins   map[string]struct{}
}

func newCORSPolicy(mode config.CORSMode, origins []string) corsPolicy {
	policy := corsPolicy{mode: mode, origins: make(map[string]struct{}, len(origins))}
	for _, origin := range origins {
		if origin == "*" {
			policy.anyOrigin = true
			continue
		}
		policy.origins[strings.ToLower(origin)] = struct{}{}
	}
	return policy
}

func (policy corsPolicy) allows(origin string) bool {
	if policy.anyOrigin {
		return true
	}
	if len(policy.origins) > 0 {
		_, listed := policy.origins[strings.ToLower(origin)]
		return listed
	}
	parsed, err := url.Parse(origin)
	return err == nil && loopback.IsHost(parsed.Hostname())
}

func handlePreflight(writer http.ResponseWriter, request *http.Request, policy corsPolicy) bool {
	requestedMethod := request.Header.Get("Access-Control-Request-Method")
	if policy.mode != config.CORSReflect || request.Method != http.MethodOptions || requestedMethod == "" {
		return false
	}
	origin := request.Header.Get("Origin")
	if !policy.allows(origin) {
		return false
	}

	applyReflectCORS(writer.Header(), origin)
	writer.Header().Set("Access-Control-Allow-Methods", requestedMethod)
	if requestedHeaders := request.Header.Get("Access-Control-Request-Headers"); requestedHeaders != "" {
		writer.Header().Set("Access-Control-Allow-Headers", requestedHeaders)
	}
	addVary(writer.Header(), "Access-Control-Request-Method")
	addVary(writer.Header(), "Access-Control-Request-Headers")
	writer.WriteHeader(http.StatusNoContent)
	return true
}

func applyResponseCORS(headers http.Header, request *http.Request, policy corsPolicy) {
	switch policy.mode {
	case config.CORSReflect:
		origin := request.Header.Get("Origin")
		if origin == "" {
			return
		}
		addVary(headers, "Origin")
		if policy.allows(origin) {
			applyReflectCORS(headers, origin)
		}
	case config.CORSOff:
		for key := range headers {
			if strings.HasPrefix(strings.ToLower(key), "access-control-") {
				headers.Del(key)
			}
		}
	}
}

func applyReflectCORS(headers http.Header, origin string) {
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
