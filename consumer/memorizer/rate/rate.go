package rate

import (
	"fmt"
)

type Rate struct {
	PreviousSample    float64
	PreviousTimestamp float64
	Sample            float64
	Timestamp         float64
}

func (r *Rate) RateCalculation(s *Rate) (float64, error) {
	if s.Timestamp == 0 || s.PreviousTimestamp == 0 {
		return 0, fmt.Errorf("rate.flush: no timestamp or previous timestamp available")
	}

	if s.Timestamp == s.PreviousTimestamp {
		return 0, fmt.Errorf("rate.flush: sample timestamp is the same as previous sample timestamp")
	}

	value := (s.Sample - s.PreviousSample) / (s.Timestamp - s.PreviousTimestamp)
	s.PreviousSample, s.PreviousTimestamp = s.Sample, s.Timestamp
	s.Sample, s.Timestamp = 0., 0.

	if value < 0 {
		return 0, fmt.Errorf("rate.flush: negative rate value detected")
	}

	return value, nil
}
