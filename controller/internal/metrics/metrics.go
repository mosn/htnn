package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	HTNNSubSystem        = "htnn"
	TranslateDurationKey = "translate_duration_seconds"
)

var (
	TranslateDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: HTNNSubSystem,
		Name:      TranslateDurationKey,
		Help:      "How long in seconds translate in a batch.",
		// minimal: 100 microseconds
		Buckets: prometheus.ExponentialBuckets(10e-5, 10, 10),
	}, []string{"controller"})

	HFPTranslateDurationObserver = TranslateDuration.WithLabelValues("httpfilterpolicy")
)

func init() {
	metrics.Registry.MustRegister(TranslateDuration)
}
