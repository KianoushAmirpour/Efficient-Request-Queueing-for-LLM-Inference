package prom

import (
	"fmt"
	"math"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"
)

type MetricBuilder struct {
	reg       promclient.Registerer
	namespace string
	firstErr  error
}

var (
	shortDurationBuckets = []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1}
	ttftDurationBuckets  = []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 7.5, 10, 15, 20, 30}
	longDurationBuckets  = []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 15, 20, 30, 45, 60, 90, 120, 180, 300}
)

func NewMetricBuilder(reg promclient.Registerer, namespace string) (*MetricBuilder, error) {
	if reg == nil {
		return nil, fmt.Errorf("prometheus registerer must not be nil")
	}
	if namespace == "" {
		namespace = "inference"
	}
	return &MetricBuilder{reg: reg, namespace: namespace}, nil
}

func validateBuckets(buckets []float64) error {
	if len(buckets) == 0 {
		return fmt.Errorf("histogram buckets must not be empty")
	}

	for i, bucket := range buckets {
		if math.IsNaN(bucket) || math.IsInf(bucket, 0) {
			return fmt.Errorf("histogram bucket %d must be finite", i)
		}
		if bucket <= 0 {
			return fmt.Errorf("histogram bucket %d must be greater than zero", i)
		}
		if i > 0 && bucket <= buckets[i-1] {
			return fmt.Errorf("histogram buckets must be strictly increasing")
		}
	}
	return nil
}

func (b *MetricBuilder) register(c promclient.Collector) {
	if b.firstErr != nil {
		return
	}
	if err := b.reg.Register(c); err != nil {
		b.firstErr = err
	}
}

func (b *MetricBuilder) err() error { return b.firstErr }

func (b *MetricBuilder) newCounterVec(name, help string, labels ...string) *promclient.CounterVec {
	c := promclient.NewCounterVec(promclient.CounterOpts{Namespace: b.namespace, Name: name, Help: help}, labels)
	b.register(c)
	return c
}

func (b *MetricBuilder) newGaugeVec(name, help string, labels ...string) *promclient.GaugeVec {
	g := promclient.NewGaugeVec(promclient.GaugeOpts{Namespace: b.namespace, Name: name, Help: help}, labels)
	b.register(g)
	return g
}

func (b *MetricBuilder) newGauge(name, help string) promclient.Gauge {
	g := promclient.NewGauge(promclient.GaugeOpts{Namespace: b.namespace, Name: name, Help: help})
	b.register(g)
	return g
}

// func (b *MetricBuilder) newHistogramVec(name, help string, labels ...string) *promclient.HistogramVec {
// 	return b.newHistogramVecWithBuckets(name, help, promclient.DefBuckets, labels...)
// }

func (b *MetricBuilder) newHistogramVecWithBuckets(name, help string, buckets []float64, labels ...string) *promclient.HistogramVec {
	if err := validateBuckets(buckets); err != nil {
		if b.firstErr == nil {
			b.firstErr = fmt.Errorf("histogram %q: %w", name, err)
		}
		return nil
	}
	h := promclient.NewHistogramVec(promclient.HistogramOpts{
		Namespace: b.namespace,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}, labels)
	b.register(h)
	return h
}

func observe(o promclient.Observer, d time.Duration) {
	if d >= 0 {
		o.Observe(d.Seconds())
	}
}
