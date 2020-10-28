package proxyproto

import (
	"bufio"
	"bytes"
	"net"
	"strconv"
	"strings"
)

const (
	crlf      = "\r\n"
	separator = " "
)

func initVersion1() *Header {
	header := new(Header)
	header.Version = 1
	// Command doesn't exist in v1
	header.Command = PROXY
	return header
}

func parseVersion1(reader *bufio.Reader) (*Header, error) {
	// Make sure we have a v1 header
	line, err := reader.ReadString('\n')
	if !strings.HasSuffix(line, crlf) {
		return nil, ErrCantReadProtocolVersionAndCommand
	}
	tokens := strings.Split(line[:len(line)-2], separator)
	if len(tokens) < 6 {
		return nil, ErrCantReadProtocolVersionAndCommand
	}

	header := initVersion1()

	// Read address family and protocol
	switch tokens[1] {
	case "TCP4":
		header.TransportProtocol = TCPv4
	case "TCP6":
		header.TransportProtocol = TCPv6
	default:
		header.TransportProtocol = UNSPEC
	}

	// Read addresses and ports
	sourceIP, err := parseV1IPAddress(header.TransportProtocol, tokens[2])
	if err != nil {
		return nil, err
	}
	destIP, err := parseV1IPAddress(header.TransportProtocol, tokens[3])
	if err != nil {
		return nil, err
	}
	sourcePort, err := parseV1PortNumber(tokens[4])
	if err != nil {
		return nil, err
	}
	destPort, err := parseV1PortNumber(tokens[5])
	if err != nil {
		return nil, err
	}
	header.SourceAddr = &net.TCPAddr{
		IP:   sourceIP,
		Port: sourcePort,
	}
	header.DestinationAddr = &net.TCPAddr{
		IP:   destIP,
		Port: destPort,
	}

	return header, nil
}

func (header *Header) formatVersion1() ([]byte, error) {
	// As of version 1, only "TCP4" ( \x54 \x43 \x50 \x34 ) for TCP over IPv4,
	// and "TCP6" ( \x54 \x43 \x50 \x36 ) for TCP over IPv6 are allowed.
	var proto string
	if header.TransportProtocol == TCPv4 {
		proto = "TCP4"
	} else if header.TransportProtocol == TCPv6 {
		proto = "TCP6"
	} else {
		// Unknown connection (short form)
		return []byte("PROXY UNKNOWN\r\n"), nil
	}

	sourceAddr := header.SourceAddr.(*net.TCPAddr)
	destAddr := header.DestinationAddr.(*net.TCPAddr)

	buf := bytes.NewBuffer(make([]byte, 0, 108))
	buf.Write(SIGV1)
	buf.WriteString(separator)
	buf.WriteString(proto)
	buf.WriteString(separator)
	buf.WriteString(sourceAddr.IP.String())
	buf.WriteString(separator)
	buf.WriteString(destAddr.IP.String())
	buf.WriteString(separator)
	buf.WriteString(strconv.Itoa(sourceAddr.Port))
	buf.WriteString(separator)
	buf.WriteString(strconv.Itoa(destAddr.Port))
	buf.WriteString(crlf)

	return buf.Bytes(), nil
}

func parseV1PortNumber(portStr string) (int, error) {
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 0 || port > 65535 {
		return 0, ErrInvalidPortNumber
	}
	return port, nil
}

func parseV1IPAddress(protocol AddressFamilyAndProtocol, addrStr string) (net.IP, error) {
	ip := net.ParseIP(addrStr)
	switch protocol {
	case TCPv4:
		ip = ip.To4()
	case TCPv6:
		ip = ip.To16()
	}
	if ip == nil {
		return nil, ErrInvalidAddress
	}
	return ip, nil
}
