package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
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

func pingIP(ip net.IP) error {
	// We have to shell out to /usr/bin/ping because creation of
	// raw ICMP endpoints requires privileges we don't have.
	cmd := exec.Command("ping", "-c1", ip.String())
	return cmd.Run()
}

func ipToMAC(ip net.IP) (string, error) {
	if mac, err := readMACForIP(ip); err == nil {
		return mac, nil
	}

	fmt.Println("pinging %v to get MAC address", ip)
	if err := pingIP(ip); err != nil {
		return "", fmt.Errorf("ping failed: %w", err)
	}

	return readMACForIP(ip)
}

func nameToMAC(name string) (string, error) {
	if hw, err := net.ParseMAC(name); err == nil {
		// Normalize the MAC we return
		return hw.String(), nil
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
