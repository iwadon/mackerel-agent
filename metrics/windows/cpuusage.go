// +build windows

package windows

import (
	"syscall"
	"unsafe"

	"github.com/mackerelio/mackerel-agent/logging"
	"github.com/mackerelio/mackerel-agent/metrics"
	"github.com/mackerelio/mackerel-agent/util/windows"
)

// CPUUsageGenerator is struct of windows api
type CPUUsageGenerator struct {
	query    syscall.Handle
	counters []*windows.CounterInfo
}

var cpuUsageLogger = logging.GetLogger("cpu.user.percentage")

// NewCPUUsageGenerator is set up windows api
func NewCPUUsageGenerator() (*CPUUsageGenerator, error) {
	g := &CPUUsageGenerator{0, nil}

	var err error
	g.query, err = windows.CreateQuery()
	if err != nil {
		cpuUsageLogger.Criticalf(err.Error())
		return nil, err
	}
	var counter *windows.CounterInfo

	counter, err = windows.CreateCounter(g.query, "cpu.user.percentage", `\Processor(_Total)\% User Time`)
	if err != nil {
		cpuUsageLogger.Criticalf(err.Error())
		return nil, err
	}
	g.counters = append(g.counters, counter)

	counter, err = windows.CreateCounter(g.query, "cpu.system.percentage", `\Processor(_Total)\% Privileged Time`)
	if err != nil {
		cpuUsageLogger.Criticalf(err.Error())
		return nil, err
	}
	g.counters = append(g.counters, counter)

	counter, err = windows.CreateCounter(g.query, "cpu.idle.percentage", `\Processor(_Total)\% Idle Time`)
	if err != nil {
		cpuUsageLogger.Criticalf(err.Error())
		return nil, err
	}
	g.counters = append(g.counters, counter)
	return g, nil
}

// Generate XXX
func (g *CPUUsageGenerator) Generate() (metrics.Values, error) {

	r, _, err := windows.PdhCollectQueryData.Call(uintptr(g.query))
	if r != 0 && err != nil {
		if r == windows.PDH_NO_DATA {
			cpuUsageLogger.Infof("this metric has not data. ")
			return nil, err
		}
		return nil, err
	}

	results := make(map[string]float64)
	for _, v := range g.counters {
		var fmtValue windows.PDH_FMT_COUNTERVALUE_DOUBLE
		r, _, err := windows.PdhGetFormattedCounterValue.Call(uintptr(v.Counter), windows.PDH_FMT_DOUBLE, uintptr(0), uintptr(unsafe.Pointer(&fmtValue)))
		if r != 0 && r != windows.PDH_INVALID_DATA {
			return nil, err
		}
		results[v.PostName] = fmtValue.DoubleValue
	}

	cpuUsageLogger.Debugf("cpuusage: %q", results)

	return results, nil
}
