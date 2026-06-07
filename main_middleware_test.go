package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipMiddleware_CompressesJSON(t *testing.T) {
	body := `{"hello":"world","items":[` + strings.Repeat(`1,`, 200) + `]}`
	handler := gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding: got %q, want gzip", got)
	}
	if got := rr.Header().Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary: got %q, want Accept-Encoding", got)
	}

	gr, err := gzip.NewReader(rr.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	decompressed, _ := io.ReadAll(gr)
	if !bytes.Equal(decompressed, []byte(body)) {
		t.Errorf("decompressed body mismatch")
	}
}

func TestGzipMiddleware_PassThroughForBinary(t *testing.T) {
	payload := bytes.Repeat([]byte{0x47, 0x40, 0x00, 0x10}, 1000)
	handler := gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp2t")
		_, _ = w.Write(payload)
	}))

	req := httptest.NewRequest("GET", "/seg.ts", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("video/mp2t should not be gzipped, got Content-Encoding=%q", got)
	}
	got, _ := io.ReadAll(rr.Body)
	if !bytes.Equal(got, payload) {
		t.Error("binary body should be passed through unchanged")
	}
}

func TestGzipMiddleware_NoAcceptEncoding(t *testing.T) {
	handler := gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"a":1}`)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("no Accept-Encoding header should mean no compression, got %q", got)
	}
}

func TestGzipMiddleware_RespectsContentTypeCharset(t *testing.T) {
	body := strings.Repeat("hello world ", 100)
	handler := gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, body)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Content-Encoding"); got != "gzip" {
		t.Errorf("text/html with charset should still gzip, got %q", got)
	}
}
