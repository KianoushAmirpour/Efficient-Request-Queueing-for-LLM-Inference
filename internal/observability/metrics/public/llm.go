package public

import "time"

// LLMGenerationMetricsRecorder records generation latency and token volume.
type LLMGenerationMetricsRecorder interface {
	ObserveTimeToFirstToken(time.Duration)
	ObserveGenerationDuration(time.Duration)
}
