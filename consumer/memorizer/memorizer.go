package memorizer

import (
	"fmt"

	"bitbucket.org/minion/metrics-system/consumer/memorizer/rate"
	"github.com/twmb/murmur3"
)

type Metric struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Value       float64  `json:"value"`
	Timestamp   int      `json:"timestamp"`
	Tags        []string `json:"tags"`
	ProjectName string   `json:"project_name"`
	Hostname    string   `json:"hostname"`
	Os          string   `json:"os"`
	UniqueId    string   `json:"unique_id"`
	Unit        string   `json:"unit"`
}

const (
	GAUGE        = "gauge"
	COUNT        = "count"
	HISTOGRAM    = "histogram"
	RATE         = "rate"
	SUMMARY      = "summary"
	METER        = "meter"
	DISTRIBUTION = "distribution"
	SET          = "set"
)

type Memorizer struct {
	metrics map[uint64]*rate.Rate
}

func NewMemorizer() *Memorizer {
	return &Memorizer{
		metrics: make(map[uint64]*rate.Rate),
	}
}

func generateContextKey(metricName string, tags []string, uid string) uint64 {
	var hash uint64
	if len(tags) > 0 {
		hash = murmur3.Sum64([]byte(metricName + fmt.Sprintf("%s", tags) + uid))
	} else {
		hash = murmur3.Sum64([]byte(metricName + uid))
	}
	return hash
}

func (m *Memorizer) ContextMemorizer(metric Metric) (Metric, error) {
	if metric.Type == "" || metric.Name == "" || metric.UniqueId == "" {
		return metric, fmt.Errorf("metric name or metric type cannot be empty")
	}
	if metric.Type == RATE {
		r := &rate.Rate{}
		if metric.Value < 0 {
			return metric, fmt.Errorf("metric value is negative value : %s", metric.Name)
		}
		var contextKey uint64
		if len(metric.Tags) > 0 {
			contextKey = generateContextKey(metric.Name, metric.Tags, metric.UniqueId)
		} else {
			contextKey = generateContextKey(metric.Name, nil, metric.UniqueId)
		}
		if mc, ok := m.metrics[contextKey]; !ok {
			m.metrics[contextKey] = &rate.Rate{
				PreviousSample:    metric.Value,
				PreviousTimestamp: float64(metric.Timestamp),
			}
			return metric, fmt.Errorf("no previous metric are not present for this metric %s", metric.Name)
		} else {
			mc.Sample = metric.Value
			mc.Timestamp = float64(metric.Timestamp)
			rate, err := r.RateCalculation(mc)
			if err != nil {
				return metric, err
			} else {
				metric.Value = rate
			}

		}

	}

	return metric, nil
}
