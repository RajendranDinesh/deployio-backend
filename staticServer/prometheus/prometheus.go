package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

var FileRequestCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "file_requests_total",
		Help: "Total number of requests per file",
	}, []string{"site", "file"},
)
