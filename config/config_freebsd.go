package config

import "path/filepath"
import "os"

var mackerelRoot = filepath.Join(os.Getenv("HOME"), "Library", "mackerel-agent")

// The default configuration for freebsd.
var DefaultConfig = &Config{
	Apibase:  "https://mackerel.io",
	Root:     mackerelRoot,
	Pidfile:  filepath.Join(mackerelRoot, "pid"),
	Conffile: filepath.Join(mackerelRoot, "mackerel-agent.conf"),
	Roles:    []string{},
	Verbose:  false,
	Connection: ConnectionConfig{
		PostMetricsDequeueDelaySeconds: 30,     // Check the metric values queue for each half minutes
		PostMetricsRetryDelaySeconds:   60,     // Wait a minute before retrying metric value posts
		PostMetricsRetryMax:            60,     // Retry up to 60 times (30s * 60 = 30min)
		PostMetricsBufferSize:          6 * 60, // Keep metric values of 6 hours span in the queue
	},
}
