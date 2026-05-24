package agent

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

func resolveOutboundIP(rawTarget string, defaultPort string) (string, error) {
	target, err := normalizeDialTarget(rawTarget, defaultPort)
	if err != nil {
		return "", err
	}

	conn, err := net.Dial("udp", target)
	if err == nil {
		defer conn.Close()

		udpAddr, ok := conn.LocalAddr().(*net.UDPAddr)
		if ok && udpAddr.IP != nil {
			return udpAddr.IP.String(), nil
		}
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP == nil || ipNet.IP.IsLoopback() {
			continue
		}

		ip4 := ipNet.IP.To4()
		if ip4 != nil {
			return ip4.String(), nil
		}
	}

	return "", errors.New("failed to resolve outbound ip")
}

func normalizeDialTarget(rawTarget string, defaultPort string) (string, error) {
	if strings.Contains(rawTarget, "://") {
		parsed, err := url.Parse(rawTarget)
		if err != nil {
			return "", err
		}

		hostPort := parsed.Host
		if hostPort == "" {
			return "", errors.New("empty server host")
		}

		if _, _, err := net.SplitHostPort(hostPort); err == nil {
			return hostPort, nil
		}

		port := defaultPort
		if parsed.Scheme == "https" {
			port = "443"
		}

		return net.JoinHostPort(hostPort, port), nil
	}

	if _, _, err := net.SplitHostPort(rawTarget); err == nil {
		return rawTarget, nil
	}

	return net.JoinHostPort(rawTarget, defaultPort), nil
}
