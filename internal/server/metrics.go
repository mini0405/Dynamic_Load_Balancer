// internal/server/metrics.go
package server

// Example: Convert CPU usage from [0..100]% to [0..1].
func NormalizeCPUUsage(rawCPU float64) float64 {
	return rawCPU / 100.0
}

// Example: Convert memory usage from [0..100]% to [0..1].
func NormalizeMemoryUsage(rawMem float64) float64 {
	return rawMem / 100.0
}

// Add additional helpers as needed...
