package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

const gzipMinSize = 256

type gzipResponseWriter struct {
	http.ResponseWriter
	gz            *gzip.Writer
	buf           []byte
	statusCode    int
	headerWritten bool
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	if g.headerWritten {
		return
	}
	g.statusCode = code
	g.ResponseWriter.WriteHeader(code)
	g.headerWritten = true
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	if g.headerWritten {
		if g.gz != nil {
			return g.gz.Write(b)
		}
		return g.ResponseWriter.Write(b)
	}
	g.buf = append(g.buf, b...)
	return len(b), nil
}

func (g *gzipResponseWriter) flush() error {
	if g.headerWritten {
		return nil
	}
	h := g.ResponseWriter.Header()
	ct := h.Get("Content-Type")
	useGzip := shouldGzip(ct) && len(g.buf) >= gzipMinSize
	if useGzip {
		h.Set("Content-Encoding", "gzip")
		h.Set("Vary", "Accept-Encoding")
		h.Del("Content-Length")
	}
	g.ResponseWriter.WriteHeader(g.statusCode)
	g.headerWritten = true
	if useGzip {
		g.gz = gzip.NewWriter(g.ResponseWriter)
		if _, err := g.gz.Write(g.buf); err != nil {
			return err
		}
		return g.gz.Close()
	}
	_, err := g.ResponseWriter.Write(g.buf)
	return err
}

func shouldGzip(contentType string) bool {
	i := strings.IndexByte(contentType, ';')
	if i >= 0 {
		contentType = contentType[:i]
	}
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	switch contentType {
	case "application/json",
		"application/javascript",
		"application/dash+xml",
		"application/vnd.apple.mpegurl",
		"application/xml",
		"image/svg+xml":
		return true
	}
	return strings.HasPrefix(contentType, "text/")
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gw := &gzipResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			buf:            make([]byte, 0, 1024),
		}
		next.ServeHTTP(gw, r)
		_ = gw.flush()
	})
}

var _ io.Writer = (*gzipResponseWriter)(nil)
