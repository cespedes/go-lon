package lon

import (
	"fmt"
	"net"
	"bytes"
	"encoding/binary"
)

type Conn struct {
	c net.Conn
}

type Cnip struct {
	Len	uint16
	Ver	uint8
	Type	uint8
	Exth	uint8
	Flags	uint8
	Vendor	uint16
	Sessid	uint32
	Seq	uint32
	Stamp	uint32
}

const CnipLen = 20

const (
	TPDU = iota
	SPDU
	AuthPDU
	APDU
)

type Lon struct {
	// Flags:
	Prior		uint8
	AltPath		uint8
	DeltaBL		uint8
	Version		uint8
	PDUFmt		uint8
	AddrFmt		uint8
	DomLen		uint8

	// Address:
	SrcSubnet	uint8
	SrcNode		uint8
	DstSubnet	uint8
	DstGroup	uint8
	DstNode		uint8
	Group		uint8
	GrpMemb		uint8
	NeuronID	uint64
	Domain		uint64
	EnclPDU		[]byte
}

type Packet struct {
	bytes	[]byte
	Cnip	Cnip
	Lon	Lon
}

func Dial(address string) (Conn, error) {
	var c Conn
	var err error
	laddr, err := net.ResolveUDPAddr("udp", ":1628")
	if err != nil {
		return c, err
	}
	daddr, err := net.ResolveUDPAddr("udp", address+":1628")
	if err != nil {
		return c, err
	}
	c.c, err = net.DialUDP("udp", laddr, daddr)
	return c, err
}

func (c Conn) Close() {
	c.c.Close()
}

func (c Conn) Read() (p Packet, err error) {
	b := make([]byte, 4096)
	i, e := c.c.Read(b)
	if e != nil {
		return p, e
	}
	b = b[0:i]
	p.bytes = b[0:i]
	r := bytes.NewReader(p.bytes)
	e = binary.Read(r, binary.BigEndian, &p.Cnip)
	p.Lon.Prior = b[CnipLen] >> 7
	p.Lon.AltPath = (b[CnipLen] >> 6) & 1
	p.Lon.DeltaBL = b[CnipLen] & 0x3F

	p.Lon.Version = b[CnipLen+1] >> 6
	p.Lon.PDUFmt = (b[CnipLen+1] >> 4) & 3
	p.Lon.AddrFmt = (b[CnipLen+1] >> 2) & 3
	p.Lon.DomLen = b[CnipLen+1] & 3

	p.Lon.SrcSubnet = b[CnipLen+2]
	p.Lon.SrcNode = b[CnipLen+3] & 0x7F

	domain_offset := 5
	switch p.Lon.AddrFmt {
	case 0:
		p.Lon.DstSubnet = b[CnipLen+4]
	case 1:
		p.Lon.DstGroup = b[CnipLen+4]
	case 2:
		p.Lon.DstSubnet = b[CnipLen+4]
		p.Lon.DstNode = b[CnipLen+5] & 0x7F
		domain_offset = 6
		if b[CnipLen+3] & 0x80 == 0 {
			domain_offset = 8
			p.Lon.Group = b[CnipLen+6]
			p.Lon.GrpMemb = b[CnipLen+7]
		}
	case 3:
		r = bytes.NewReader(b[CnipLen+5:CnipLen+11])
		e = binary.Read(r, binary.BigEndian, &p.Lon.NeuronID)
	}
	switch p.Lon.DomLen {
	case 0:
		p.Lon.Domain = 0
		p.Lon.EnclPDU = b[CnipLen+domain_offset:]
	case 1:
		p.Lon.Domain = uint64(b[CnipLen+domain_offset])
		p.Lon.EnclPDU = b[CnipLen+domain_offset+1:]
	case 2:
		p.Lon.Domain = (uint64(b[CnipLen+domain_offset]) << 16) |
				(uint64(b[CnipLen+domain_offset+1]) << 8) |
				(uint64(b[CnipLen+domain_offset+2]))
		p.Lon.EnclPDU = b[CnipLen+domain_offset+3:]
	case 3:
		p.Lon.Domain = (uint64(b[CnipLen+domain_offset]) << 40) |
				(uint64(b[CnipLen+domain_offset+1]) << 32) |
				(uint64(b[CnipLen+domain_offset+2]) << 24) |
				(uint64(b[CnipLen+domain_offset+3]) << 16) |
				(uint64(b[CnipLen+domain_offset+4]) << 8) |
				(uint64(b[CnipLen+domain_offset+5]))
		p.Lon.EnclPDU = b[CnipLen+domain_offset+6:]
	}
	return p, e
}

func (p Packet) Src() (s string) {
	s = fmt.Sprintf("%d,%d,%d", p.Lon.Domain, p.Lon.SrcSubnet, p.Lon.SrcNode)
	return
}

func (p Packet) Dst() (d string) {
	switch p.Lon.AddrFmt {
	case 0: // DstSubnet
		return fmt.Sprintf("%d,*", p.Lon.DstSubnet)
	case 1: // DstGroup
		return fmt.Sprintf("g=%d", p.Lon.DstGroup)
	case 2:
		if p.bytes[CnipLen+3] & 0x80 == 0x80 { // DstSubnet + DstNode
			return fmt.Sprintf("%d,%d", p.Lon.DstSubnet, p.Lon.DstNode)
		} else { // DstSubnet + DstNode + Group + GrpMemb
			return fmt.Sprintf("%d,%d,g=%d,m=%d", p.Lon.DstSubnet, p.Lon.DstNode, p.Lon.Group, p.Lon.GrpMemb)
		}
	case 3: // DstSubnet + DstNode
		return fmt.Sprintf("%d,%d,n=%012x", p.Lon.DstSubnet, p.Lon.DstNode, p.Lon.NeuronID)
	default:
		return "???"
	}
}

func (p Packet) Debug() string {
	return fmt.Sprintf("%v %v %v %s -> %s", p.bytes, p.Cnip, p.Lon, p.Src(), p.Dst())
}

func APDU_String(b []byte) string {
	return fmt.Sprintf("dt=%d,size=%d", b[0], len(b)-1)
}

func (p Packet) String() string {
	var addrs string = fmt.Sprintf("%s -> %s", p.Src(), p.Dst())
	var pdu string

	switch p.Lon.PDUFmt {
	case TPDU:
		switch p.Lon.EnclPDU[0] & 0x70 >> 4 {
			case 0:
				pdu = "TPDU ACKD(0) " + APDU_String(p.Lon.EnclPDU[1:])
			case 1:
				pdu = "TPDU UnACKD_TPR(1) " + APDU_String(p.Lon.EnclPDU[1:])
			case 2:
				pdu = "TPDU ACK(2)"
			case 4:
				pdu = "TPDU REMINDER(4)"
			case 5:
				pdu = "TPDU REM/MSG(5)"
			default:
				pdu = fmt.Sprintf("TPDU ???(%d)", p.Lon.EnclPDU[0] & 0x70 >> 4)
		}
	case SPDU:
		switch p.Lon.EnclPDU[0] & 0x70 >> 4 {
			case 0:
				pdu = "SPDU REQUEST(0) " + APDU_String(p.Lon.EnclPDU[1:])
			case 2:
				pdu = "SPDU RESPONSE(2) " + APDU_String(p.Lon.EnclPDU[1:])
			case 4:
				pdu = "SPDU REMINDER(4)"
			case 5:
				pdu = "SPDU REM/MSG(5)"
			default:
				pdu = fmt.Sprintf("SPDU ???(%d)", p.Lon.EnclPDU[0] & 0x70 >> 4)
		}
	case AuthPDU:
		switch p.Lon.EnclPDU[0] & 0x30 >> 4 {
			case 0:
				pdu = "AuthPDU CHALLENGE(0)"
			case 2:
				pdu = "AuthPDU REPLY(2)"
			default:
				pdu = fmt.Sprintf("AuthPDU ???(%d)", p.Lon.EnclPDU[0] & 0x30 >> 4)
		}
	case APDU:
		pdu = "APDU " + APDU_String(p.Lon.EnclPDU)
	}

	return addrs + " " + pdu
}
