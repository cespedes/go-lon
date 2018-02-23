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

type Packet struct {
	bytes	[]byte
	Cnip	Cnip
	Lon	[]byte
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

func (c Conn) Read(b []byte) (i int, e error, cnip Cnip) {
	i, e = c.c.Read(b)
	if e != nil {
		return i, e, cnip
	}
	r := bytes.NewReader(b)
	e = binary.Read(r, binary.BigEndian, &cnip)
	return i, e, cnip
}

func (c Conn) Read2() (p Packet, err error) {
	b := make([]byte, 4096)
	i, e := c.c.Read(b)
	if e != nil {
		return p, e
	}
	p.bytes = b[0:i]
	r := bytes.NewReader(p.bytes)
	e = binary.Read(r, binary.BigEndian, &p.Cnip)
	p.Lon = b[20:i]
	return p, e
}

func (p Packet) String() string {
	return fmt.Sprintf("%v %v", p.Cnip, p.Lon)
}
