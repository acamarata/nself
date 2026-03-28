package docker

import (
	"fmt"
	"net"
	"time"
)

// ReservedPorts lists ports used by the nSelf stack that must be checked
// for conflicts before starting services.
var ReservedPorts = []int{
	80, 443, 5432, 8080, 4000, 6379,
	9000, 9001, 7700, 3021, 1025, 8025,
	3008, 5000,
}

// PortConflict records whether a specific port is already in use.
type PortConflict struct {
	Port  int
	InUse bool
}

// CheckPort probes a single TCP port on localhost using DialTimeout.
// It returns true when the port is in use (connection succeeded) and
// false when the port is available (connection refused or timed out).
func CheckPort(port int) (bool, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		// Connection refused or timed out — port is available.
		return false, nil
	}
	conn.Close()
	// Connection succeeded — something is listening.
	return true, nil
}

// CheckAllPorts probes every port in the slice and returns only the
// entries where the port is already in use. A nil slice means no
// conflicts were found.
func CheckAllPorts(ports []int) ([]PortConflict, error) {
	var conflicts []PortConflict
	for _, p := range ports {
		inUse, err := CheckPort(p)
		if err != nil {
			return nil, fmt.Errorf("checking port %d: %w", p, err)
		}
		if inUse {
			conflicts = append(conflicts, PortConflict{Port: p, InUse: true})
		}
	}
	return conflicts, nil
}
