package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	// Register prometheus metrics.
	Register()
}

const (
	namespace = "engine"
	subsystem = "daemon"
)

var (
	// ImagePullSummary records the summary of pulling image latency.
	ImagePullSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "image_pull_latency_microseconds",
			Help:      "Latency in microseconds to pull a image.",
		},
		[]string{"image"},
	)

	ContainerActionsCounter        = newLabelCounter("container_actions_counter", "The number of container operations", "action")
	ContainerSuccessActionsCounter = newLabelCounter("container_success_actions_counter", "The number of container success operations", "action")
	ImageActionsCounter            = newLabelCounter("image_actions_counter", "The number of image operations", "action")
	ImageSuccessActionsCounter     = newLabelCounter("image_success_actions_counter", "The number of image success operations", "action")
	ContainerActionsTimer          = newLabelTimer("container_actions", "The number of seconds it takes to process each container action", "action")
	ImageActionsTimer              = newLabelTimer("image_actions", "The number of seconds it takes to process each image action", "action")

	EngineVersion = newLabelGauge("engine", "The version and commit information for the engine process",
		"commit",
	)
)

var registerMetrics sync.Once

// Register all metrics.
func Register() {
	// Register the metrics.
	registerMetrics.Do(func() {
		prometheus.MustRegister(ImagePullSummary)
		prometheus.MustRegister(EngineVersion)
		prometheus.MustRegister(ContainerActionsCounter)
		prometheus.MustRegister(ContainerSuccessActionsCounter)
		prometheus.MustRegister(ImageActionsCounter)
		prometheus.MustRegister(ImageSuccessActionsCounter)
		prometheus.MustRegister(ContainerActionsTimer)
		prometheus.MustRegister(ImageActionsTimer)
	})
}

// SinceInMicroseconds gets the time since the specified start in microseconds.
func SinceInMicroseconds(start time.Time) float64 {
	return float64(time.Since(start).Nanoseconds() / time.Microsecond.Nanoseconds())
}

func newLabelCounter(name, help string, labels ...string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        fmt.Sprintf("%s_%s", name, Total),
			Help:        help,
			ConstLabels: nil,
		},
		labels)
}

func newLabelGauge(name, help string, labels ...string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        fmt.Sprintf("%s_%s", name, Unit("info")),
			Help:        help,
			ConstLabels: nil,
		}, labels)
}

func newLabelTimer(name, help string, labels ...string) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        fmt.Sprintf("%s_%s", name, Seconds),
			Help:        help,
			ConstLabels: nil,
		}, labels)
}
