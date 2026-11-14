// Package rtp implements VP8 RTP packetization per RFC 7741.
package rtp

import "encoding/binary"

// MaxPayloadSize is the default max RTP payload (fits in typical MTU).
const MaxPayloadSize = 1200

// Packet is a single RTP packet with VP8 payload.
type Packet struct {
	Header  Header
	Payload []byte
}

// Header is a simplified RTP header.
type Header struct {
	SequenceNumber uint16
	Timestamp      uint32
	SSRC           uint32
	Marker         bool // last packet of frame
	PayloadType    uint8
}

// Packetizer splits VP8 frames into RTP packets per RFC 7741.
type Packetizer struct {
	ssrc    uint32
	pt      uint8
	seq     uint16
	mtu     int
	clockHz uint32
}

// NewPacketizer creates a VP8 RTP packetizer.
func NewPacketizer(ssrc uint32, payloadType uint8, clockRate uint32, mtu int) *Packetizer {
	if mtu <= 0 {
		mtu = MaxPayloadSize
	}
	return &Packetizer{
		ssrc:    ssrc,
		pt:      payloadType,
		mtu:     mtu,
		clockHz: clockRate,
	}
}

// Packetize splits a VP8 frame into RTP packets.
// keyframe indicates if this is a keyframe (affects VP8 payload descriptor).
func (p *Packetizer) Packetize(frame []byte, timestamp uint32, keyframe bool) []Packet {
	// VP8 payload descriptor overhead: 1 byte minimum.
	maxChunk := p.mtu - 1 // 1 byte for VP8 payload descriptor
	if maxChunk <= 0 {
		maxChunk = 1
	}

	var packets []Packet
	offset := 0
	first := true

	for offset < len(frame) {
		end := offset + maxChunk
		if end > len(frame) {
			end = len(frame)
		}
		chunk := frame[offset:end]
		last := end >= len(frame)

		// VP8 payload descriptor (1 byte, no extensions).
		// Bit 0 (X): 0 (no extension)
		// Bit 1 (R): 0 (reserved)
		// Bit 2 (N): 1 if non-reference frame (we always set 0)
		// Bit 3 (S): 1 if start of VP8 partition
		// Bit 4 (PartID): 0
		var desc byte
		if first {
			desc |= 0x10 // S=1 (start of partition)
		}
		_ = keyframe // keyframe info is in the VP8 bitstream itself

		payload := make([]byte, 1+len(chunk))
		payload[0] = desc
		copy(payload[1:], chunk)

		pkt := Packet{
			Header: Header{
				SequenceNumber: p.seq,
				Timestamp:      timestamp,
				SSRC:           p.ssrc,
				Marker:         last,
				PayloadType:    p.pt,
			},
			Payload: payload,
		}
		packets = append(packets, pkt)
		p.seq++
		offset = end
		first = false
	}
	return packets
}

// Depacketizer reassembles VP8 frames from RTP packets.
type Depacketizer struct {
	buf       []byte
	timestamp uint32
	started   bool
}

// NewDepacketizer creates a VP8 RTP depacketizer.
func NewDepacketizer() *Depacketizer {
	return &Depacketizer{}
}

// Push adds an RTP packet. Returns the complete frame when marker bit is set.
// Returns nil if frame is not yet complete.
func (d *Depacketizer) Push(pkt Packet) []byte {
	desc := pkt.Payload[0]
	data := pkt.Payload[1:]

	// S bit = start of new partition.
	if desc&0x10 != 0 {
		d.buf = d.buf[:0]
		d.timestamp = pkt.Header.Timestamp
		d.started = true
	}

	if !d.started {
		return nil
	}

	d.buf = append(d.buf, data...)

	if pkt.Header.Marker {
		frame := make([]byte, len(d.buf))
		copy(frame, d.buf)
		d.buf = d.buf[:0]
		d.started = false
		return frame
	}
	return nil
}

// Marshal serializes an RTP packet to wire format.
func (pkt *Packet) Marshal() []byte {
	hdrLen := 12
	buf := make([]byte, hdrLen+len(pkt.Payload))
	// V=2, P=0, X=0, CC=0.
	buf[0] = 0x80
	// M + PT.
	buf[1] = pkt.Header.PayloadType & 0x7F
	if pkt.Header.Marker {
		buf[1] |= 0x80
	}
	binary.BigEndian.PutUint16(buf[2:4], pkt.Header.SequenceNumber)
	binary.BigEndian.PutUint32(buf[4:8], pkt.Header.Timestamp)
	binary.BigEndian.PutUint32(buf[8:12], pkt.Header.SSRC)
	copy(buf[12:], pkt.Payload)
	return buf
}

// Unmarshal parses an RTP packet from wire format.
func Unmarshal(data []byte) (Packet, error) {
	var pkt Packet
	if len(data) < 12 {
		return pkt, errShort
	}
	pkt.Header.Marker = data[1]&0x80 != 0
	pkt.Header.PayloadType = data[1] & 0x7F
	pkt.Header.SequenceNumber = binary.BigEndian.Uint16(data[2:4])
	pkt.Header.Timestamp = binary.BigEndian.Uint32(data[4:8])
	pkt.Header.SSRC = binary.BigEndian.Uint32(data[8:12])
	// Skip CSRC and extensions for simplicity.
	pkt.Payload = data[12:]
	return pkt, nil
}

type rtpError string

func (e rtpError) Error() string { return string(e) }

const errShort = rtpError("rtp: packet too short")
