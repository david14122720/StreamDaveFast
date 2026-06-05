package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeProcessedTree builds a temp dir with the given relative file paths
// under "processed/<videoName>/" and returns the temp dir + the video name.
// The fake server in each test serves files from this tree.
func fakeProcessedTree(t *testing.T, videoName string, files map[string]string) (rootDir string) {
	t.Helper()
	rootDir = t.TempDir()
	videoDir := filepath.Join(rootDir, "processed", videoName)
	if err := os.MkdirAll(videoDir, 0o755); err != nil {
		t.Fatalf("mkdir videoDir: %v", err)
	}
	for name, body := range files {
		full := filepath.Join(videoDir, name)
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	return rootDir
}

// serveFromRoot wraps handleDASHFiles with a server that is rooted at
// the given temp directory, so "."+URL.Path resolves to files under it.
// We do this by chdir-ing into the temp dir for the duration of the test
// (handleDASHFiles builds filePath as "." + rawPath, so it picks up cwd).
func serveFromRoot(t *testing.T, rootDir string) *httptest.Server {
	t.Helper()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("chdir %s: %v", rootDir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	ts := httptest.NewServer(http.HandlerFunc(handleDASHFiles))
	t.Cleanup(ts.Close)
	return ts
}

func TestHLSManifestMIME(t *testing.T) {
	root := fakeProcessedTree(t, "demo", map[string]string{
		"master.m3u8": "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:6\n#EXTINF:6.0,\nseg-000.ts\n#EXT-X-ENDLIST\n",
	})
	ts := serveFromRoot(t, root)

	resp, err := http.Get(ts.URL + "/processed/demo/master.m3u8")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/vnd.apple.mpegurl" {
		t.Errorf("Content-Type: got %q, want %q", got, "application/vnd.apple.mpegurl")
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.HasPrefix(string(body), "#EXTM3U") {
		t.Errorf("body should start with #EXTM3U, got %q", string(body)[:min(20, len(body))])
	}
}

func TestHLSSegmentMIME(t *testing.T) {
	root := fakeProcessedTree(t, "demo", map[string]string{
		"seg-000.ts": "\x47\x40\x00\x10", // TS sync byte + minimal PAT-looking prefix
	})
	ts := serveFromRoot(t, root)

	resp, err := http.Get(ts.URL + "/processed/demo/seg-000.ts")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "video/mp2t" {
		t.Errorf("Content-Type: got %q, want %q", got, "video/mp2t")
	}
	// First request is a miss, so the response must come from disk and be
	// re-cached. We don't assert X-Cache here because both HIT and MISS are
	// acceptable; we DO assert the body matches the file we wrote.
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 4 || body[0] != 0x47 {
		t.Errorf("body should be 4 bytes starting with 0x47 (TS sync), got %v", body)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
