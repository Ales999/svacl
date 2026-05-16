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

// ParseSVIAclFile parses a single Cisco config file and returns all SVI info with ACLs.
func ParseSVIAclFile(filePath string) ([]SVIAclInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var results []SVIAclInfo
	var hostname string

	for scanner.Scan() {
		line := scanner.Text()

		// Check for null bytes (binary file detection).
		if strings.ContainsRune(line, '\x00') {
			return nil, fmt.Errorf("%s contains null bytes — not a valid text file", filePath)
		}

		// Extract hostname once from the entire file.
		if hostname == "" && strings.HasPrefix(line, "hostname") {
			hostname = strings.TrimSpace(line[8:])
		}

		// Detect start of an SVI interface block.
		if !strings.HasPrefix(line, "interface ") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 || !strings.HasPrefix(parts[1], "Vlan") {
			continue
		}

		svi := SVIAclInfo{
			Hostname: hostname,
			VlanName: parts[1],
		}

		// Scan body lines until a non-indented line or file end.
		for scanner.Scan() {
			bodyLine := scanner.Text()

			if strings.ContainsRune(bodyLine, '\x00') {
				return nil, fmt.Errorf("%s contains null bytes — not a valid text file", filePath)
			}

			// End of interface block when line doesn't start with space or tab.
			if len(bodyLine) == 0 || (bodyLine[0] != ' ' && bodyLine[0] != '\t') {
				break
			}

			stripped := strings.TrimSpace(bodyLine)
			if stripped == "" {
				continue
			}

			// Parse shutdown (not inside description).
			if strings.Contains(stripped, "shutdown") && !strings.Contains(stripped, "description") {
				svi.Shutdown = true
			}

			// Parse IP address.
			if strings.HasPrefix(stripped, "ip address ") {
				subParts := strings.Fields(stripped)
				if len(subParts) >= 4 {
					ipStr := subParts[2]
					maskStr := subParts[3]

					maskOctets := strings.Split(maskStr, ".")
					if len(maskOctets) == 4 {
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
						svi.IPAddr = ipStr + "/" + fmt.Sprintf("%d", prefixLen)
					}
				}
			}

			// Parse VRF.
			if strings.HasPrefix(stripped, "vrf forwarding ") || strings.HasPrefix(stripped, "ip vrf forwarding ") {
				vParts := strings.Fields(stripped)
				if len(vParts) >= 3 {
					svi.VRF = vParts[len(vParts)-1]
				}
			}

			// Parse ACL.
			if strings.HasPrefix(stripped, "ip access-group ") {
				aParts := strings.Fields(stripped)
				if len(aParts) >= 3 {
					aclName := aParts[2]
					if len(aParts) >= 4 {
						direction := aParts[3]
						switch direction {
						case "in":
							svi.ACLIn = aclName
						case "out":
							svi.ACLOut = aclName
						}
					}
				}
			}
		}

		results = append(results, svi)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return results, nil
}
