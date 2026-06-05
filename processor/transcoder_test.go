package processor

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestTranscodeVideoProducesHLSManifest is the first end-to-end test for
// the transcoder. It generates a 5-second synthetic source with ffmpeg's
// lavfi testsrc (no fixture file needed), runs TranscodeVideo, and asserts
// that both the DASH manifest AND the HLS manifest exist on disk.
//
// Skipped when ffmpeg is not in PATH (CI environments may not have it).
func TestTranscodeVideoProducesHLSManifest(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not in PATH; skipping integration test")
	}

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "testsrc.mp4")
	outputDir := filepath.Join(dir, "out")

	// Generate a 5-second 320x240 30fps test pattern. The encoder settings
	// match the production defaults (libx264, yuv420p) so the HLS remux
	// pass has a valid H.264 stream to wrap.
	genArgs := []string{
		"-y",
		"-f", "lavfi",
		"-i", "testsrc=duration=5:size=320x240:rate=30",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-preset", "ultrafast",
		inputPath,
	}
	genCmd := exec.Command("ffmpeg", genArgs...)
	genCmd.Stderr = os.Stderr
	if err := genCmd.Run(); err != nil {
		t.Fatalf("generate test input: %v", err)
	}

	// Run the production transcoder (DASH + HLS passes).
	result, err := TranscodeVideo(inputPath, outputDir)
	if err != nil {
		t.Fatalf("TranscodeVideo: %v", err)
	}
	if result == nil {
		t.Fatal("TranscodeVideo returned nil result")
	}

	// DASH artifacts (existing behavior, regression guard).
	manifestPath := filepath.Join(outputDir, "manifest.mpd")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("DASH manifest.mpd missing: %v", err)
	}

	// HLS artifacts (PR #1 acceptance criteria).
	hlsManifest := filepath.Join(outputDir, "master.m3u8")
	if _, err := os.Stat(hlsManifest); err != nil {
		t.Errorf("HLS master.m3u8 missing: %v", err)
	}

	// At least one .ts segment must exist; otherwise the HLS playlist is
	// empty and iOS Safari will fail to start playback.
	tsEntries, err := filepath.Glob(filepath.Join(outputDir, "seg-*.ts"))
	if err != nil {
		t.Fatalf("glob seg-*.ts: %v", err)
	}
	if len(tsEntries) == 0 {
		t.Error("no seg-*.ts segments produced; HLS playlist would be empty")
	}

	// TranscodeResult.HLSManifest should be populated for API consumers.
	if result.HLSManifest == "" {
		t.Error("TranscodeResult.HLSManifest is empty")
	}
	if result.HLSManifest != hlsManifest {
		t.Errorf("TranscodeResult.HLSManifest = %q, want %q", result.HLSManifest, hlsManifest)
	}
}
