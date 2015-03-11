package swim

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
)

func TestIP4Addr(t *testing.T) {
	ip := []byte{1, 2, 3, 4}
	testIPAddr(t, IP4Addr, ip, "")
}

func TestIP6Addr(t *testing.T) {
	ip := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	testIPAddr(t, IP6Addr, ip, "lo1")
}

func testIPAddr(t *testing.T, addrType AddrType, ip []byte, zone string) {
	ipAddr := &net.IPAddr{IP: net.IP(ip), Zone: zone}
	addr := &Addr{ipAddr}
	zb := bytes.NewBufferString(zone).Bytes()
	data := append(append([]byte{byte(addrType)}, ip...), zb...)

	testMarshalBinary(t, addr, data)
	testUnmarshalBinary(t, addr, data)

	// expect new addr
	if addr.Addr == ipAddr {
		t.Fatal("expected UnmarshalBinary to create new net.Addr")
	}
	subject, ok := addr.Addr.(*net.IPAddr)
	if !ok {
		t.Fatalf("expected IPAddr got %v", addr.Addr)
	}
	testEqualBytes(t, subject.IP, ip)
	if subject.Zone != zone {
		t.Fatalf("expected zone %v got %v", zone, subject.Zone)
	}
}

func TestTCP4Addr(t *testing.T) {
	ip := []byte{1, 2, 3, 4}
	testTCPAddr(t, TCP4Addr, ip, 234, "")
}

func TestTCP6Addr(t *testing.T) {
	ip := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	testTCPAddr(t, TCP6Addr, ip, 234, "lo1")
}

func testTCPAddr(t *testing.T, addrType AddrType, ip []byte, port int, zone string) {
	ipAddr := &net.TCPAddr{IP: net.IP(ip), Port: port, Zone: zone}
	addr := &Addr{ipAddr}

	var buf bytes.Buffer
	port16 := uint16(port)
	buf.Write([]byte{byte(addrType)})
	buf.Write(ip)
	binary.Write(&buf, binary.BigEndian, &port16)
	buf.WriteString(zone)
	data := buf.Bytes()

	testMarshalBinary(t, addr, data)
	testUnmarshalBinary(t, addr, data)

	// expect new addr
	if addr.Addr == ipAddr {
		t.Fatal("expected UnmarshalBinary to create new net.Addr")
	}
	subject, ok := addr.Addr.(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected TCPAddr got %v", addr.Addr)
	}
	testEqualBytes(t, subject.IP, ip)
	if subject.Zone != zone {
		t.Fatalf("expected zone %v got %v", zone, subject.Zone)
	}
	if subject.Port != port {
		t.Fatalf("expected port %v got %v", subject.Port, port)
	}
}

func TestUDP4Addr(t *testing.T) {
	ip := []byte{1, 2, 3, 4}
	testUDPAddr(t, UDP4Addr, ip, 234, "")
}

func TestUDP6Addr(t *testing.T) {
	ip := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	testUDPAddr(t, UDP6Addr, ip, 234, "lo1")
}

func testUDPAddr(t *testing.T, addrType AddrType, ip []byte, port int, zone string) {
	ipAddr := &net.UDPAddr{IP: net.IP(ip), Port: port, Zone: zone}
	addr := &Addr{ipAddr}

	var buf bytes.Buffer
	port16 := uint16(port)
	buf.Write([]byte{byte(addrType)})
	buf.Write(ip)
	binary.Write(&buf, binary.BigEndian, &port16)
	buf.WriteString(zone)
	data := buf.Bytes()

	testMarshalBinary(t, addr, data)
	testUnmarshalBinary(t, addr, data)

	// expect new addr
	if addr.Addr == ipAddr {
		t.Fatal("expected UnmarshalBinary to create new net.Addr")
	}
	subject, ok := addr.Addr.(*net.UDPAddr)
	if !ok {
		t.Fatalf("expected UDPAddr got %v", addr.Addr)
	}
	testEqualBytes(t, subject.IP, ip)
	if subject.Zone != zone {
		t.Fatalf("expected zone %v got %v", zone, subject.Zone)
	}
	if subject.Port != port {
		t.Fatalf("expected port %v got %v", subject.Port, port)
	}
}
