// +build darwin

package darwin

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/mackerelio/mackerel-agent/logging"
	"github.com/mackerelio/mackerel-agent/metrics"
)

/*
MemoryGenerator collect memory usage

`memory.{metric}`: using memory size retrieved from `vm_stat` and `sysctl vm.swapusage`

metric = "total", "free", "used", "cached", "active", "inactive", "swap_total", "swap_used", "swap_free"

graph: stacks `memory.{metric}`
*/
type MemoryGenerator struct {
}

/* vm_stat sample
% vm_stat
Mach Virtual Memory Statistics: (page size of 4096 bytes)
Pages free:                               72827.
Pages active:                           2154445.
Pages inactive:                         1511468.
Pages speculative:                         8107.
Pages throttled:                              0.
Pages wired down:                        446975.
Pages purgeable:                         383371.
"Translation faults":                  97589077.
Pages copy-on-write:                    3305869.
Pages zero filled:                     50848672.
Pages reactivated:                         1999.
Pages purged:                           2496610.
File-backed pages:                       677870.
Anonymous pages:                        2996150.
Pages stored in compressor:                   0.
Pages occupied by compressor:                 0.
Decompressions:                               0.
Compressions:                                 0.
Pageins:                                6333901.
Pageouts:                                   353.
Swapins:                                      0.
Swapouts:                                     0.
*/

/* sysctl vm.swapusage sample
% sysctl vm.swapusage
vm.swapusage: total = 1024.00M  used = 2.50M  free = 1021.50M  (encrypted)
*/

var memoryLogger = logging.GetLogger("metrics.memory")
var statReg = regexp.MustCompile(`^(.+):\s+([0-9]+)\.$`)
var swapReg = regexp.MustCompile(`([0-9]+(?:\.[0-9]+))?M[^0-9]*([0-9]+(?:\.[0-9]+)?)M[^0-9]*([0-9]+(?:\.[0-9]+)?)M`)

// Generate generate metrics values
func (g *MemoryGenerator) Generate() (metrics.Values, error) {
	outBytes, err := exec.Command("vm_stat").Output()
	if err != nil {
		memoryLogger.Errorf("Failed (skip these metrics): %s", err)
		return nil, err
	}
	out := string(outBytes)
	lines := strings.Split(out, "\n")

	stats := map[string]int64{}
	for _, line := range lines {
		if matches := statReg.FindStringSubmatch(line); matches != nil {
			v, _ := strconv.ParseInt(matches[2], 0, 64)
			// `vm_stat` returns the page sise of 4096 bytes
			stats[matches[1]] = v * 4096
		}
	}

	wired := stats["Pages wired down"]
	compressed := stats["Pages occupied by compressor"]

	cached := stats["Pages purgeable"] + stats["File-backed pages"]
	active := stats["Pages active"]
	inactive := stats["Pages inactive"]
	used := wired + compressed + active + inactive + stats["Pages speculative"] - cached
	free := stats["Pages free"]
	total := used + cached + free

	ret := map[string]float64{
		"memory.total":    float64(total),
		"memory.used":     float64(used),
		"memory.cached":   float64(cached),
		"memory.free":     float64(free),
		"memory.active":   float64(active),
		"memory.inactive": float64(inactive),
	}

	outBytes, err = exec.Command("sysctl", "vm.swapusage").Output()
	if err != nil {
		memoryLogger.Errorf("Failed (skip swap metrics): %s", err)
	} else {
		if matches := swapReg.FindStringSubmatch(string(outBytes)); matches != nil {
			t, _ := strconv.ParseFloat(matches[1], 64)
			// swap_used are calculated at server, so don't send it
			// u, _ := strconv.ParseFloat(matches[2], 64)
			f, _ := strconv.ParseFloat(matches[3], 64)
			ret["memory.swap_total"] = t * 1024 * 1024
			ret["memory.swap_free"] = f * 1024 * 1024
		}
	}
	return metrics.Values(ret), nil
}
