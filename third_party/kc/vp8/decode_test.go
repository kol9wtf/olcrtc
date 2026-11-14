package vp8

import (
	"encoding/binary"
	"os"
	"os/exec"
	"testing"
)

// TestDecodeFFmpegInterFrame generates a multi-frame IVF with ffmpeg and
// verifies our decoder can at least parse the keyframe.
func TestDecodeFFmpegInterFrame(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found")
	}

	// Generate a 3-frame VP8 IVF using ffmpeg from a test pattern.
	ivfPath := t.TempDir() + "/test.ivf"
	cmd := exec.Command("ffmpeg", "-y",
		"-f", "lavfi", "-i", "testsrc=duration=0.1:size=64x64:rate=30",
		"-c:v", "libvpx", "-g", "30", "-f", "ivf", ivfPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ffmpeg encode: %v\n%s", err, out)
	}

	// Read IVF and decode first frame (keyframe).
	data, err := os.ReadFile(ivfPath)
	if err != nil {
		t.Fatal(err)
	}

	// Parse IVF header.
	if string(data[0:4]) != "DKIF" {
		t.Fatal("not IVF")
	}
	width := int(binary.LittleEndian.Uint16(data[12:14]))
	height := int(binary.LittleEndian.Uint16(data[14:16]))

	// First frame.
	frameSize := int(binary.LittleEndian.Uint32(data[32:36]))
	frameData := data[44 : 44+frameSize]

	dec := NewDecoder()
	frame, err := dec.Decode(frameData)
	if err != nil {
		t.Fatalf("decode keyframe: %v", err)
	}
	if frame.Width != width || frame.Height != height {
		t.Fatalf("dimensions: got %dx%d want %dx%d", frame.Width, frame.Height, width, height)
	}
	t.Logf("decoded ffmpeg keyframe: %dx%d, Y=%d bytes", width, height, len(frame.Y))
}
