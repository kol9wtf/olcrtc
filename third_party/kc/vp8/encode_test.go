package vp8

import (
	"encoding/binary"
	"os"
	"os/exec"
	"testing"
)

// TestEncodeFFmpegDecode encodes a test frame and verifies ffmpeg can decode it.
func TestEncodeFFmpegDecode(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found, skipping integration test")
	}

	// Create a simple 64x64 grayscale test pattern.
	const w, h = 64, 64
	gray := make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if (x/8+y/8)%2 == 0 {
				gray[y*w+x] = 255
			}
		}
	}

	enc := NewEncoder(w, h, 4)
	frame, err := enc.Encode(gray)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	// Wrap in IVF container for ffmpeg.
	ivf := makeIVF(frame, w, h)

	// Write to temp file.
	tmp, err := os.CreateTemp("", "kc-test-*.ivf")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(ivf); err != nil {
		t.Fatalf("write: %v", err)
	}
	tmp.Close()

	// Try to decode with ffmpeg.
	out := tmp.Name() + ".yuv"
	defer os.Remove(out)

	cmd := exec.Command("ffmpeg", "-y", "-i", tmp.Name(), "-f", "rawvideo", "-pix_fmt", "yuv420p", out)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ffmpeg decode failed: %v\n%s", err, output)
	}

	// Check output file exists and has expected size.
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	expectedSize := int64(w*h + w*h/2) // YUV420
	if info.Size() != expectedSize {
		t.Fatalf("output size=%d want=%d", info.Size(), expectedSize)
	}
}

// TestRoundTrip verifies our encoder -> our decoder produces correct pixels.
func TestRoundTrip(t *testing.T) {
	t.Parallel()
	const w, h = 64, 64
	gray := make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if (x/8+y/8)%2 == 0 {
				gray[y*w+x] = 255
			}
		}
	}

	enc := NewEncoder(w, h, 0) // qp=0 for lossless-ish
	frame, err := enc.Encode(gray)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	dec := NewDecoder()
	f, err := dec.Decode(frame)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if f.Width != w || f.Height != h {
		t.Fatalf("dimensions: got %dx%d want %dx%d", f.Width, f.Height, w, h)
	}

	// Check Y plane matches (with tolerance for quantization).
	out := f.Grayscale()
	maxDiff := 0
	for i := range gray {
		d := int(gray[i]) - int(out[i])
		if d < 0 {
			d = -d
		}
		if d > maxDiff {
			maxDiff = d
		}
	}
	// At qp=0, VP8 still has some quantization error but should be small.
	if maxDiff > 10 {
		t.Fatalf("max pixel diff=%d (too high for qp=0)", maxDiff)
	}
	t.Logf("round-trip max pixel diff: %d", maxDiff)
}

// makeIVF wraps a single VP8 frame in an IVF container.
func makeIVF(frame []byte, width, height int) []byte {
	buf := make([]byte, 32+12+len(frame))
	// IVF header (32 bytes).
	copy(buf[0:4], "DKIF")
	binary.LittleEndian.PutUint16(buf[4:6], 0)  // version
	binary.LittleEndian.PutUint16(buf[6:8], 32) // header size
	copy(buf[8:12], "VP80")                     // codec
	binary.LittleEndian.PutUint16(buf[12:14], uint16(width))
	binary.LittleEndian.PutUint16(buf[14:16], uint16(height))
	binary.LittleEndian.PutUint32(buf[16:20], 120) // fps num
	binary.LittleEndian.PutUint32(buf[20:24], 1)   // fps den
	binary.LittleEndian.PutUint32(buf[24:28], 1)   // frame count
	binary.LittleEndian.PutUint32(buf[28:32], 0)   // unused

	// Frame header (12 bytes).
	binary.LittleEndian.PutUint32(buf[32:36], uint32(len(frame)))
	binary.LittleEndian.PutUint64(buf[36:44], 0) // timestamp

	// Frame data.
	copy(buf[44:], frame)
	return buf
}
