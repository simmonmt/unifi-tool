package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func readMACForIP(ip net.IP) (string, error) {
	f, err := os.Open("/proc/net/arp")
	if err != nil {
		return "", err
	}
	defer f.Close()

	ipStr := ip.String()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if fields[0] == ipStr {
			mac := fields[3]
			if _, err := net.ParseMAC(mac); err != nil {
				return "", fmt.Errorf(`found invalid MAC "%v" for %v: %w`,
					mac, ipStr, err)
			}

			return mac, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("no MAC found for %v", ip)
}

func ipToMAC(ip net.IP) (string, error) {
	if mac, err := readMACForIP(ip); err == nil {
		return mac, nil
	}

	// It wasn't there. Ping it.

	return readMACForIP(ip)
}

func nameToMAC(name string) (string, error) {
	if _, err := net.ParseMAC(name); err == nil {
		return name, nil
	}

	if ip := net.ParseIP(name); ip != nil {
		return ipToMAC(ip)
	}

	ips, err := net.LookupIP(name)
	if err != nil {
		return "", err
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("no IP addresses found")
	}

	return ipToMAC(ips[0])
}
