package public

import "time"

type HTTPMetricsRecorder interface {
	ObserveHTTPRequest(method, route string, status int)
	ObserveHTTPRequestDuration(method, route string, d time.Duration)
	IncHTTPInFlight(method, route string)
	DecHTTPInFlight(method, route string)
}
