package agent

import (
	"time"

	"github.com/mackerelio/mackerel-agent/config"
	"github.com/mackerelio/mackerel-agent/mackerel"
	"github.com/mackerelio/mackerel-agent/metrics"
)

// Agent XXX
type Agent struct {
	MetricsGenerators []metrics.Generator
	PluginGenerators  []metrics.PluginGenerator
}

// MetricsResult XXX
type MetricsResult struct {
	Created time.Time
	Values  metrics.Values
}

func (agent *Agent) collectMetrics(collectedTime time.Time) *MetricsResult {
	generators := agent.MetricsGenerators
	for _, g := range agent.PluginGenerators {
		generators = append(generators, g)
	}
	result := generateValues(generators)
	values := <-result
	return &MetricsResult{Created: collectedTime, Values: values}
}

// Watch XXX
func (agent *Agent) Watch() chan *MetricsResult {

	metricsResult := make(chan *MetricsResult)
	ticker := make(chan time.Time)

	go func() {
		c := time.Tick(1 * time.Second)

		last := time.Now()
		ticker <- last // sends tick once at first

		for t := range c {
			// Fire an event at 0 second per minute.
			// Because ticks may not be accurate,
			// fire an event if t - last is more than 1 minute
			if t.Second()%int(config.PostMetricsInterval.Seconds()) == 0 || t.After(last.Add(config.PostMetricsInterval)) {
				last = t
				ticker <- t
			}
		}
	}()

	const collectMetricsWorkerMax = 3

	go func() {
		// Start collectMetrics concurrently
		// so that it does not prevent runnnig next collectMetrics.
		sem := make(chan uint, collectMetricsWorkerMax)
		for tickedTime := range ticker {
			ti := tickedTime
			sem <- 1
			go func() {
				metricsResult <- agent.collectMetrics(ti)
				<-sem
			}()
		}
	}()

	return metricsResult
}

// InitPluginGenerators XXX
func (agent *Agent) InitPluginGenerators(api *mackerel.API) {
	payloads := []mackerel.CreateGraphDefsPayload{}

	for _, g := range agent.PluginGenerators {
		p, err := g.PrepareGraphDefs(api)
		if err != nil {
			logger.Debugf("Failed to fetch meta information from plugin %s (non critical); seems that this plugin does not have meta information: %s", g, err)
		}
		if p != nil {
			payloads = append(payloads, p...)
		}
	}

	if len(payloads) > 0 {
		err := api.CreateGraphDefs(payloads)
		if err != nil {
			logger.Errorf("Failed to create graphdefs: %s", err)
		}
	}
}
