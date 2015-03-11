package swim

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	"net"
)

var InvalidAddressError = errors.New("invalid address")

type AddrType int

const (
	_ AddrType = iota
	IP4Addr
	IP6Addr
	TCP4Addr
	TCP6Addr
	UDP4Addr
	UDP6Addr
	UnixAddr
	UnixGramAddr
	UnixPacketAddr
	UserAddr
)

type Addr struct {
	net.Addr
}

func (a Addr) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer

	switch a := a.Addr.(type) {
	case *net.IPAddr:

		// write address type
		if len(a.IP) == 4 { // IPv4
			buf.WriteByte(byte(IP4Addr))
		} else { // IPv6
			buf.WriteByte(byte(IP6Addr))
		}

		// write IP address
		buf.Write([]byte(a.IP))

		// write IPv6 zone
		if len(a.Zone) > 0 {
			buf.WriteString(a.Zone)
		}

	case *net.TCPAddr:

		// write address type
		if len(a.IP) == 4 { // IPv4
			buf.WriteByte(byte(TCP4Addr))
		} else { // IPv6
			buf.WriteByte(byte(TCP6Addr))
		}

		// write IP address
		buf.Write([]byte(a.IP))

		// write port
		port := uint16(a.Port)
		binary.Write(&buf, binary.BigEndian, port)

		// write IPv6 zone
		if len(a.Zone) > 0 {
			buf.WriteString(a.Zone)
		}

	case *net.UDPAddr:

		// write address type
		if len(a.IP) == 4 { // IPv4
			buf.WriteByte(byte(UDP4Addr))
		} else { // IPv6
			buf.WriteByte(byte(UDP6Addr))
		}

		// write IP address
		buf.Write([]byte(a.IP))

		// write port
		port := uint16(a.Port)
		binary.Write(&buf, binary.BigEndian, port)

		// write IPv6 zone
		if len(a.Zone) > 0 {
			buf.WriteString(a.Zone)
		}

	case *net.UnixAddr:

		// write address type
		switch a.Net {
		case "unix":
			buf.WriteByte(byte(UnixAddr))
		case "unixgram":
			buf.WriteByte(byte(UnixGramAddr))
		case "unixpacket":
			buf.WriteByte(byte(UnixPacketAddr))
		}

		// write address
		buf.WriteString(a.Name)

	case encoding.BinaryMarshaler:
		if data, err = a.MarshalBinary(); err == nil {
			data = append([]byte{byte(UserAddr)}, data...)
		}
		return

	default:
		return nil, InvalidAddressError
	}

	return buf.Bytes(), nil
}

func (a *Addr) UnmarshalBinary(data []byte) error {
	t, data := AddrType(data[0]), data[1:]
	switch t {
	case IP4Addr:
		a.Addr = &net.IPAddr{IP: net.IP(data[0:4])}

	case IP6Addr:
		a.Addr = &net.IPAddr{
			IP:   net.IP(data[0:16]),
			Zone: bytes.NewBuffer(data[16:]).String(),
		}

	case TCP4Addr:

		// read port
		var port uint16
		portReader := bytes.NewReader(data[4:6])
		if err := binary.Read(portReader, binary.BigEndian, &port); err != nil {
			return err
		}

		// set address
		a.Addr = &net.TCPAddr{
			IP:   net.IP(data[0:4]),
			Port: int(port),
			Zone: bytes.NewBuffer(data[6:]).String(),
		}

	case TCP6Addr:

		// read port
		var port uint16
		portReader := bytes.NewReader(data[16:18])
		if err := binary.Read(portReader, binary.BigEndian, &port); err != nil {
			return err
		}

		// set address
		a.Addr = &net.TCPAddr{
			IP:   net.IP(data[0:16]),
			Port: int(port),
			Zone: bytes.NewBuffer(data[18:]).String(),
		}

	case UDP4Addr:

		// read port
		var port uint16
		portReader := bytes.NewReader(data[4:6])
		if err := binary.Read(portReader, binary.BigEndian, &port); err != nil {
			return err
		}

		// set address
		a.Addr = &net.UDPAddr{
			IP:   net.IP(data[0:4]),
			Port: int(port),
			Zone: bytes.NewBuffer(data[6:]).String(),
		}

	case UDP6Addr:

		// read port
		var port uint16
		portReader := bytes.NewReader(data[16:18])
		if err := binary.Read(portReader, binary.BigEndian, &port); err != nil {
			return err
		}

		// set address
		a.Addr = &net.UDPAddr{
			IP:   net.IP(data[0:16]),
			Port: int(port),
			Zone: bytes.NewBuffer(data[18:]).String(),
		}

	case UnixAddr:
		a.Addr = &net.UnixAddr{
			Name: bytes.NewBuffer(data).String(),
			Net:  "unix",
		}

	case UnixGramAddr:
		a.Addr = &net.UnixAddr{
			Name: bytes.NewBuffer(data).String(),
			Net:  "unixgram",
		}

	case UnixPacketAddr:
		a.Addr = &net.UnixAddr{
			Name: bytes.NewBuffer(data).String(),
			Net:  "unixpacket",
		}

	case UserAddr:
		if addr, ok := a.Addr.(encoding.BinaryUnmarshaler); !ok {
			return InvalidAddressError
		} else {
			return addr.UnmarshalBinary(data)
		}

	default:
		return InvalidAddressError
	}

	return nil
}
