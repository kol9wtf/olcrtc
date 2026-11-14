// Package ivf implements the IVF container format for VP8 frames.
package ivf

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Header is the 32-byte IVF file header.
type Header struct {
	Width       uint16
	Height      uint16
	TimebaseNum uint32
	TimebaseDen uint32
	FrameCount  uint32
}

// Writer writes VP8 frames into an IVF container.
type Writer struct {
	w          io.WriteSeeker
	hdr        Header
	frameCount uint32
}

// NewWriter creates an IVF writer. Writes the header immediately.
func NewWriter(w io.WriteSeeker, width, height, fps int) (*Writer, error) {
	iw := &Writer{
		w: w,
		hdr: Header{
			Width:       uint16(width),
			Height:      uint16(height),
			TimebaseNum: uint32(fps),
			TimebaseDen: 1,
		},
	}
	if err := iw.writeHeader(); err != nil {
		return nil, err
	}
	return iw, nil
}

// WriteFrame writes a single VP8 frame with the given timestamp.
func (iw *Writer) WriteFrame(data []byte, pts uint64) error {
	var buf [12]byte
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(data)))
	binary.LittleEndian.PutUint64(buf[4:12], pts)
	if _, err := iw.w.Write(buf[:]); err != nil {
		return fmt.Errorf("ivf: write frame header: %w", err)
	}
	if _, err := iw.w.Write(data); err != nil {
		return fmt.Errorf("ivf: write frame data: %w", err)
	}
	iw.frameCount++
	return nil
}

// Close finalizes the IVF file by updating the frame count in the header.
func (iw *Writer) Close() error {
	if _, err := iw.w.Seek(24, io.SeekStart); err != nil {
		return fmt.Errorf("ivf: seek: %w", err)
	}
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], iw.frameCount)
	if _, err := iw.w.Write(buf[:]); err != nil {
		return fmt.Errorf("ivf: update frame count: %w", err)
	}
	return nil
}

func (iw *Writer) writeHeader() error {
	var buf [32]byte
	copy(buf[0:4], "DKIF")
	binary.LittleEndian.PutUint16(buf[4:6], 0)  // version
	binary.LittleEndian.PutUint16(buf[6:8], 32) // header size
	copy(buf[8:12], "VP80")
	binary.LittleEndian.PutUint16(buf[12:14], iw.hdr.Width)
	binary.LittleEndian.PutUint16(buf[14:16], iw.hdr.Height)
	binary.LittleEndian.PutUint32(buf[16:20], iw.hdr.TimebaseNum)
	binary.LittleEndian.PutUint32(buf[20:24], iw.hdr.TimebaseDen)
	binary.LittleEndian.PutUint32(buf[24:28], 0) // frame count (updated on close)
	binary.LittleEndian.PutUint32(buf[28:32], 0) // unused
	_, err := iw.w.Write(buf[:])
	if err != nil {
		return fmt.Errorf("ivf: write header: %w", err)
	}
	return nil
}

// Reader reads VP8 frames from an IVF container.
type Reader struct {
	r   io.Reader
	Hdr Header
}

// NewReader creates an IVF reader. Reads and validates the header.
func NewReader(r io.Reader) (*Reader, error) {
	var buf [32]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return nil, fmt.Errorf("ivf: read header: %w", err)
	}
	if string(buf[0:4]) != "DKIF" {
		return nil, fmt.Errorf("ivf: invalid signature")
	}
	if string(buf[8:12]) != "VP80" {
		return nil, fmt.Errorf("ivf: not VP8 codec")
	}
	hdr := Header{
		Width:       binary.LittleEndian.Uint16(buf[12:14]),
		Height:      binary.LittleEndian.Uint16(buf[14:16]),
		TimebaseNum: binary.LittleEndian.Uint32(buf[16:20]),
		TimebaseDen: binary.LittleEndian.Uint32(buf[20:24]),
		FrameCount:  binary.LittleEndian.Uint32(buf[24:28]),
	}
	return &Reader{r: r, Hdr: hdr}, nil
}

// ReadFrame reads the next frame. Returns io.EOF when no more frames.
func (ir *Reader) ReadFrame() (data []byte, pts uint64, err error) {
	var buf [12]byte
	if _, err := io.ReadFull(ir.r, buf[:]); err != nil {
		return nil, 0, fmt.Errorf("ivf: read frame header: %w", err)
	}
	size := binary.LittleEndian.Uint32(buf[0:4])
	pts = binary.LittleEndian.Uint64(buf[4:12])
	data = make([]byte, size)
	if _, err := io.ReadFull(ir.r, data); err != nil {
		return nil, 0, fmt.Errorf("ivf: read frame data: %w", err)
	}
	return data, pts, nil
}
