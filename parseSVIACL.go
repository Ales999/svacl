package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type SVIAclInfo struct {
	Hostname string
	VlanName string
	IPAddr   string
	VRF      string
	Shutdown bool
	ACLIn    string
	ACLOut   string
}

// parseVlan extracts "Vlan2006" from "interface Vlan2006".
func parseVlan(line string) string {
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// parseAclName extracts ACL name from " ip access-group vlan2006_in in" -> "vlan2006_in".
func parseAclName(line string) string {
	parts := strings.Fields(line)
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

// parseIpAddr extracts "172.24.2006.1/24" from " ip address 172.24.2006.1 255.255.255.0".
func parseIpAddr(line string) string {
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return ""
	}
	ipStr := parts[2]
	maskStr := parts[3]

	maskOctets := strings.Split(maskStr, ".")
	if len(maskOctets) != 4 {
		return ""
	}

	prefixLen := 0
	for _, o := range maskOctets {
		var val int
		fmt.Sscanf(o, "%d", &val)
		for i := 7; i >= 0; i-- {
			if val&(1<<i) != 0 {
				prefixLen++
			} else {
				break
			}
		}
	}

	return ipStr + "/" + fmt.Sprintf("%d", prefixLen)
}

// ParseSVIAclFile parses a single Cisco config file and returns all SVI info with ACLs.
func ParseSVIAclFile(filePath string) ([]SVIAclInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var allLines []string
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var results []SVIAclInfo
	var hostname string

	for i, line := range allLines {
		// Extract hostname once.
		if hostname == "" && strings.HasPrefix(line, "hostname") {
			hostname = strings.TrimSpace(line[9:])
		}

		// Detect SVI interface start.
		if !strings.HasPrefix(line, "interface ") {
			continue
		}
		ifaceName := strings.Fields(line)
		if len(ifaceName) < 2 || !strings.HasPrefix(ifaceName[1], "Vlan") {
			continue
		}

		svi := SVIAclInfo{
			Hostname: hostname,
			VlanName: parseVlan(line),
		}

		// Collect body lines (next 20 lines that start with whitespace).
		startIdx := i + 1
		endIdx := len(allLines)
		if startIdx+20 < endIdx {
			endIdx = startIdx + 20
		}

		for j := startIdx; j < endIdx; j++ {
			bodyLine := allLines[j]

			// End of interface block when line doesn't start with space.
			if len(bodyLine) == 0 || (bodyLine[0] != ' ' && bodyLine[0] != '\t') {
				break
			}

			stripped := strings.TrimSpace(bodyLine)
			if stripped == "" {
				continue
			}

			// Check for shutdown (not in description).
			if strings.Contains(stripped, "shutdown") && !strings.Contains(stripped, "description") {
				svi.Shutdown = true
			}

			// Parse IP address.
			if strings.HasPrefix(stripped, "ip address ") {
				svi.IPAddr = parseIpAddr(stripped)
			}

			// Parse VRF.
			if stripped == "vrf forwarding" || stripped == "ip vrf forwarding" {
				// skip — need the next word
			} else if strings.HasPrefix(stripped, "vrf forwarding ") {
				parts := strings.Fields(stripped)
				if len(parts) >= 3 {
					svi.VRF = parts[2]
				}
			} else if strings.HasPrefix(stripped, "ip vrf forwarding ") {
				parts := strings.Fields(stripped)
				if len(parts) >= 4 {
					svi.VRF = parts[3]
				}
			}

			// Parse ACL.
			if strings.HasPrefix(stripped, "ip access-group") {
				aclName := parseAclName(stripped)
				if aclName == "" {
					continue
				}
				if strings.HasSuffix(stripped, " in") || len(strings.Fields(stripped)) >= 4 && strings.Fields(stripped)[3] == "in" {
					svi.ACLIn = aclName
				}
				if strings.HasSuffix(stripped, " out") || len(strings.Fields(stripped)) >= 4 && strings.Fields(stripped)[3] == "out" {
					svi.ACLOut = aclName
				}
			}
		}

		results = append(results, svi)
	}

	return results, nil
}
