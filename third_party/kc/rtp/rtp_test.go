package rtp

import (
	"bytes"
	"testing"
)

func TestPacketizeDepacketize(t *testing.T) {
	t.Parallel()
	// Create a fake VP8 frame.
	frame := make([]byte, 5000)
	for i := range frame {
		frame[i] = byte(i)
	}

	p := NewPacketizer(0x12345678, 96, 90000, 1200)
	packets := p.Packetize(frame, 1000, true)

	if len(packets) < 5 {
		t.Fatalf("expected at least 5 packets, got %d", len(packets))
	}

	// Last packet should have marker.
	if !packets[len(packets)-1].Header.Marker {
		t.Fatal("last packet missing marker")
	}
	// First packet should have S bit.
	if packets[0].Payload[0]&0x10 == 0 {
		t.Fatal("first packet missing S bit")
	}

	// Depacketize.
	dp := NewDepacketizer()
	var result []byte
	for _, pkt := range packets {
		if f := dp.Push(pkt); f != nil {
			result = f
		}
	}
	if result == nil {
		t.Fatal("no frame reassembled")
	}
	if !bytes.Equal(result, frame) {
		t.Fatalf("frame mismatch: got %d bytes want %d", len(result), len(frame))
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	t.Parallel()
	pkt := Packet{
		Header: Header{
			SequenceNumber: 42,
			Timestamp:      90000,
			SSRC:           0xDEADBEEF,
			Marker:         true,
			PayloadType:    96,
		},
		Payload: []byte{0x10, 0xAA, 0xBB},
	}
	wire := pkt.Marshal()
	got, err := Unmarshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if got.Header.SequenceNumber != 42 {
		t.Fatalf("seq=%d", got.Header.SequenceNumber)
	}
	if got.Header.Timestamp != 90000 {
		t.Fatalf("ts=%d", got.Header.Timestamp)
	}
	if got.Header.SSRC != 0xDEADBEEF {
		t.Fatalf("ssrc=%x", got.Header.SSRC)
	}
	if !got.Header.Marker {
		t.Fatal("marker not set")
	}
	if !bytes.Equal(got.Payload, pkt.Payload) {
		t.Fatal("payload mismatch")
	}
}
