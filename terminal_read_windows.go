//go:build windows

package main

import "time"

// readWithTimeout is a no-op on Windows where raw fd polling is not
// available. Graphics protocol probes will fall back to basic rendering.
func readWithTimeout(_ int, _ []byte, _ time.Duration) (int, error) {
	return 0, nil
}
