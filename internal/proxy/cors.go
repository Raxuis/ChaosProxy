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

func (p corsPolicy) allows(origin string) bool {
	if p.anyOrigin {
		return true
	}
	if len(p.origins) > 0 {
		_, listed := p.origins[strings.ToLower(origin)]
		return listed
	}
	parsed, err := url.Parse(origin)
	return err == nil && loopback.IsHost(parsed.Hostname())
}

func handlePreflight(w http.ResponseWriter, r *http.Request, policy corsPolicy) bool {
	requestedMethod := r.Header.Get("Access-Control-Request-Method")
	if policy.mode != config.CORSReflect || r.Method != http.MethodOptions || requestedMethod == "" {
		return false
	}
	origin := r.Header.Get("Origin")
	if !policy.allows(origin) {
		return false
	}

	applyReflectCORS(w.Header(), origin)
	w.Header().Set("Access-Control-Allow-Methods", requestedMethod)
	if requestedHeaders := r.Header.Get("Access-Control-Request-Headers"); requestedHeaders != "" {
		w.Header().Set("Access-Control-Allow-Headers", requestedHeaders)
	}
	addVary(w.Header(), "Access-Control-Request-Method")
	addVary(w.Header(), "Access-Control-Request-Headers")
	w.WriteHeader(http.StatusNoContent)
	return true
}

func applyResponseCORS(headers http.Header, r *http.Request, policy corsPolicy) {
	switch policy.mode {
	case config.CORSReflect:
		origin := r.Header.Get("Origin")
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
