package faults

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

const mutateSource = `{"user":{"name":"Ada","email":"ada@example.com","admin":true,"visits":42},` +
	`"items":[{"id":1,"price":9.5,"secret":"a"},{"id":2,"price":3,"tags":["new"]}],"count":2}`

func TestMutateOperations(t *testing.T) {
	tests := []struct {
		name      string
		operation config.MutationConfig
		want      string
	}{
		{
			name:      "nullify object key",
			operation: config.MutationConfig{Op: "nullify", Path: "user.email"},
			want: `{"user":{"name":"Ada","email":null,"admin":true,"visits":42},` +
				`"items":[{"id":1,"price":9.5,"secret":"a"},{"id":2,"price":3,"tags":["new"]}],"count":2}`,
		},
		{
			name:      "empty string",
			operation: config.MutationConfig{Op: "empty", Path: "user.name"},
			want: `{"user":{"name":"","email":"ada@example.com","admin":true,"visits":42},` +
				`"items":[{"id":1,"price":9.5,"secret":"a"},{"id":2,"price":3,"tags":["new"]}],"count":2}`,
		},
		{
			name:      "empty array",
			operation: config.MutationConfig{Op: "empty", Path: "items"},
			want:      `{"user":{"name":"Ada","email":"ada@example.com","admin":true,"visits":42},"items":[],"count":2}`,
		},
		{
			name:      "empty number and boolean",
			operation: config.MutationConfig{Op: "empty", Path: "user.*"},
			want: `{"user":{"name":"","email":"","admin":false,"visits":0},` +
				`"items":[{"id":1,"price":9.5,"secret":"a"},{"id":2,"price":3,"tags":["new"]}],"count":2}`,
		},
		{
			name:      "inflate array",
			operation: config.MutationConfig{Op: "inflate", Path: "items.1.tags", Factor: 3},
			want: `{"user":{"name":"Ada","email":"ada@example.com","admin":true,"visits":42},` +
				`"items":[{"id":1,"price":9.5,"secret":"a"},{"id":2,"price":3,"tags":["new","new","new"]}],"count":2}`,
		},
		{
			name:      "stretch string",
			operation: config.MutationConfig{Op: "stretch", Path: "user.name", Factor: 3},
			want: `{"user":{"name":"AdaAdaAda","email":"ada@example.com","admin":true,"visits":42},` +
				`"items":[{"id":1,"price":9.5,"secret":"a"},{"id":2,"price":3,"tags":["new"]}],"count":2}`,
		},
		{
			name:      "drop object key",
			operation: config.MutationConfig{Op: "drop", Path: "user.email"},
			want: `{"user":{"name":"Ada","admin":true,"visits":42},` +
				`"items":[{"id":1,"price":9.5,"secret":"a"},{"id":2,"price":3,"tags":["new"]}],"count":2}`,
		},
		{
			name:      "drop array index",
			operation: config.MutationConfig{Op: "drop", Path: "items.0"},
			want:      `{"user":{"name":"Ada","email":"ada@example.com","admin":true,"visits":42},"items":[{"id":2,"price":3,"tags":["new"]}],"count":2}`,
		},
		{
			name:      "nullify through array wildcard",
			operation: config.MutationConfig{Op: "nullify", Path: "items.*.price"},
			want: `{"user":{"name":"Ada","email":"ada@example.com","admin":true,"visits":42},` +
				`"items":[{"id":1,"price":null,"secret":"a"},{"id":2,"price":null,"tags":["new"]}],"count":2}`,
		},
		{
			name:      "drop through array wildcard",
			operation: config.MutationConfig{Op: "drop", Path: "items.*.secret"},
			want: `{"user":{"name":"Ada","email":"ada@example.com","admin":true,"visits":42},` +
				`"items":[{"id":1,"price":9.5},{"id":2,"price":3,"tags":["new"]}],"count":2}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, injections := mutateResponse(t, config.MutateConfig{Probability: 1, MaxBytes: 1 << 20, Operations: []config.MutationConfig{test.operation}},
				"application/json", mutateSource, true)
			assertSameJSON(t, readAll(t, response), test.want)
			if !hasFault(injections, "mutate") {
				t.Errorf("injections = %+v, want a mutate fault", injections)
			}
		})
	}
}

func TestMutateRecordsNoOpsAndKeepsBodyIntact(t *testing.T) {
	response, injections := mutateResponse(t, config.MutateConfig{Probability: 1, MaxBytes: 1 << 20, Operations: []config.MutationConfig{
		{Op: "nullify", Path: "user.phone"},
		{Op: "drop", Path: "items.9"},
		{Op: "stretch", Path: "count", Factor: 2},
	}}, "application/json", mutateSource, true)

	if body := readAll(t, response); body != mutateSource {
		t.Fatalf("body = %s, want the original bytes", body)
	}
	if hasFault(injections, "mutate") {
		t.Errorf("injections = %+v, want no mutate fault", injections)
	}
	for _, want := range []string{"mutate nullify user.phone matched nothing", "mutate drop items.9 matched nothing", "mutate stretch count matched nothing"} {
		if !hasDetail(injections, want) {
			t.Errorf("details %+v do not contain %q", injections, want)
		}
	}
}

func TestMutateRewritesLengthAndPreservesValues(t *testing.T) {
	source := `{"id":12345678901234567890,"html":"<b>hi</b>","user":{"email":"x"}}`
	response, _ := mutateResponse(t, config.MutateConfig{Probability: 1, MaxBytes: 1 << 20, Operations: []config.MutationConfig{
		{Op: "nullify", Path: "user.email"},
	}}, "application/json; charset=utf-8", source, true)
	response.Header.Set("ETag", `"stale"`)

	body := readAll(t, response)
	if response.ContentLength != int64(len(body)) || response.Header.Get("Content-Length") != strconv.Itoa(len(body)) {
		t.Errorf("Content-Length = %d/%q, want %d", response.ContentLength, response.Header.Get("Content-Length"), len(body))
	}
	if !strings.Contains(body, "12345678901234567890") || !strings.Contains(body, "<b>hi</b>") {
		t.Errorf("body = %s, want exact numbers and unescaped HTML", body)
	}
}

func TestMutateRemovesStaleETag(t *testing.T) {
	var injections []Injection
	ctx := newFaultContext()
	ctx.Emit = func(injection Injection) { injections = append(injections, injection) }
	response := &http.Response{
		Header:        http.Header{"Content-Type": []string{"application/json"}, "Etag": []string{`"v1"`}},
		Body:          io.NopCloser(strings.NewReader(mutateSource)),
		ContentLength: int64(len(mutateSource)),
	}
	fault := newMutateFault(config.MutateConfig{Probability: 1, MaxBytes: 1 << 20, Operations: []config.MutationConfig{{Op: "drop", Path: "count"}}})
	if err := fault.After(ctx, response); err != nil {
		t.Fatalf("After() unexpected error: %v", err)
	}
	if response.Header.Get("ETag") != "" {
		t.Errorf("ETag = %q, want it removed after mutation", response.Header.Get("ETag"))
	}
}

func TestMutateGates(t *testing.T) {
	drop := []config.MutationConfig{{Op: "drop", Path: "count"}}
	tests := []struct {
		name        string
		cfg         config.MutateConfig
		contentType string
		encoding    string
		body        string
		knownLength bool
		mutated     bool
		detail      string
	}{
		{name: "not JSON", cfg: config.MutateConfig{Probability: 1, MaxBytes: 1 << 20, Operations: drop}, contentType: "text/html", body: mutateSource, knownLength: true},
		{name: "JSON suffix", cfg: config.MutateConfig{Probability: 1, MaxBytes: 1 << 20, Operations: drop}, contentType: "application/problem+json; charset=utf-8", body: mutateSource, knownLength: true, mutated: true},
		{name: "compressed", cfg: config.MutateConfig{Probability: 1, MaxBytes: 1 << 20, Operations: drop}, contentType: "application/json", encoding: "gzip", body: mutateSource, knownLength: true, detail: "gzip-encoded body"},
		{name: "declared length too large", cfg: config.MutateConfig{Probability: 1, MaxBytes: 10, Operations: drop}, contentType: "application/json", body: mutateSource, knownLength: true, detail: "exceeds max_bytes 10"},
		{name: "streamed body too large", cfg: config.MutateConfig{Probability: 1, MaxBytes: 10, Operations: drop}, contentType: "application/json", body: mutateSource, detail: "body exceeds max_bytes 10"},
		{name: "invalid JSON", cfg: config.MutateConfig{Probability: 1, MaxBytes: 1 << 20, Operations: drop}, contentType: "application/json", body: `{"user":`, knownLength: true, detail: "not valid JSON"},
		{name: "probability zero", cfg: config.MutateConfig{Probability: 0, MaxBytes: 1 << 20, Operations: drop}, contentType: "application/json", body: mutateSource, knownLength: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, injections := mutateResponseWithEncoding(t, test.cfg, test.contentType, test.encoding, test.body, test.knownLength)
			body := readAll(t, response)
			if test.mutated == (body == test.body) {
				t.Errorf("body = %s, want mutated=%t", body, test.mutated)
			}
			if hasFault(injections, "mutate") != test.mutated {
				t.Errorf("injections = %+v, want mutate fault=%t", injections, test.mutated)
			}
			if test.detail != "" && !hasDetail(injections, test.detail) {
				t.Errorf("details %+v do not contain %q", injections, test.detail)
			}
		})
	}
}

func mutateResponse(t *testing.T, cfg config.MutateConfig, contentType, body string, knownLength bool) (*http.Response, []Injection) {
	t.Helper()
	return mutateResponseWithEncoding(t, cfg, contentType, "", body, knownLength)
}

func mutateResponseWithEncoding(t *testing.T, cfg config.MutateConfig, contentType, encoding, body string, knownLength bool) (*http.Response, []Injection) {
	t.Helper()

	var injections []Injection
	ctx := newFaultContext()
	ctx.Emit = func(injection Injection) { injections = append(injections, injection) }
	response := &http.Response{
		Header:        http.Header{"Content-Type": []string{contentType}},
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: -1,
	}
	if encoding != "" {
		response.Header.Set("Content-Encoding", encoding)
	}
	if knownLength {
		response.ContentLength = int64(len(body))
		response.Header.Set("Content-Length", strconv.Itoa(len(body)))
	}
	if err := newMutateFault(cfg).After(ctx, response); err != nil {
		t.Fatalf("After() unexpected error: %v", err)
	}
	return response, injections
}

func readAll(t *testing.T, response *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}

func assertSameJSON(t *testing.T, got, want string) {
	t.Helper()
	decode := func(text string) any {
		decoder := json.NewDecoder(bytes.NewReader([]byte(text)))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			t.Fatalf("decode %s: %v", text, err)
		}
		return value
	}
	if !reflect.DeepEqual(decode(got), decode(want)) {
		t.Fatalf("JSON = %s\nwant   %s", got, want)
	}
}

func hasFault(injections []Injection, fault string) bool {
	for _, injection := range injections {
		if injection.Fault == fault {
			return true
		}
	}
	return false
}

func hasDetail(injections []Injection, detail string) bool {
	for _, injection := range injections {
		if strings.Contains(injection.Detail, detail) {
			return true
		}
	}
	return false
}
