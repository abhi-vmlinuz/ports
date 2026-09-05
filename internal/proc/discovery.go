package proc

import (
	"os"
	"sort"

	"ports/internal/model"
)

// Discoverer coordinates socket discovery, PID mapping, and process metadata enrichment.
type Discoverer struct {
	netScanner     *NetScanner
	procPath       string
	processScanner *ProcessScanner
}

// NewDiscoverer creates a Discoverer.
func NewDiscoverer(procPath string) *Discoverer {
	if procPath == "" {
		procPath = "/proc"
	}
	return &Discoverer{
		netScanner:     NewNetScanner(procPath + "/net"),
		procPath:       procPath,
		processScanner: NewProcessScanner(procPath),
	}
}

// DiscoverAll finds all listening TCP and bound UDP ports and enriches them with process metadata.
func (d *Discoverer) DiscoverAll() ([]model.PortRecord, error) {
	sockets, err := d.netScanner.ScanSockets()
	if err != nil {
		return nil, err
	}

	targetInodes := make(map[uint64]struct{}, len(sockets))
	for _, s := range sockets {
		if s.Inode > 0 {
			targetInodes[s.Inode] = struct{}{}
		}
	}

	inodeMap, err := ScanInodesToPIDs(d.procPath, targetInodes)
	if err != nil {
		return nil, err
	}

	isRoot := os.Geteuid() == 0
	var records []model.PortRecord

	for _, s := range sockets {
		record := model.PortRecord{
			Port:     s.LocalPort,
			Protocol: s.Protocol,
			Address:  s.LocalIP,
		}

		pid, hasPID := inodeMap[s.Inode]
		if hasPID && pid > 0 {
			record.PID = pid
			pInfo := d.processScanner.GetProcess(pid)

			record.Process = pInfo.Name
			if record.Process == "" {
				record.Process = "<unknown>"
			}

			if pInfo.UID >= 0 {
				uid := pInfo.UID
				record.UID = &uid
			}
			if pInfo.User != "" {
				userStr := pInfo.User
				record.User = &userStr
			}
			if pInfo.CWD != "" {
				cwdStr := pInfo.CWD
				record.CWD = &cwdStr
			}
			if pInfo.Cmdline != "" {
				cmdStr := pInfo.Cmdline
				record.Command = &cmdStr
			}
			if pInfo.UptimeSeconds > 0 {
				up := pInfo.UptimeSeconds
				record.UptimeSeconds = &up
			}
		} else {
			// PID could not be mapped
			sockUID := s.SocketUID
			record.UID = &sockUID
			sockUser := ResolveUsername(sockUID)
			record.User = &sockUser

			if !isRoot && sockUID != os.Geteuid() {
				record.Process = "<permission denied>"
			} else {
				record.Process = "<unknown>"
			}
		}

		records = append(records, record)
	}

	// Sort records according to Section 15 of spec:
	// 1. numeric port ascending
	// 2. protocol
	// 3. PID
	// 4. address
	sort.Slice(records, func(i, j int) bool {
		if records[i].Port != records[j].Port {
			return records[i].Port < records[j].Port
		}
		if records[i].Protocol != records[j].Protocol {
			return records[i].Protocol < records[j].Protocol
		}
		if records[i].PID != records[j].PID {
			return records[i].PID < records[j].PID
		}
		return records[i].Address < records[j].Address
	})

	return records, nil
}

// DiscoverPort returns all records matching a specific port.
func (d *Discoverer) DiscoverPort(port uint16) ([]model.PortRecord, error) {
	all, err := d.DiscoverAll()
	if err != nil {
		return nil, err
	}

	var matched []model.PortRecord
	for _, r := range all {
		if r.Port == port {
			matched = append(matched, r)
		}
	}

	return matched, nil
}
