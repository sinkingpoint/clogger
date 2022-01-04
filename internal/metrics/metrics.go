package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	MessagesProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "clogger",
		Name:      "messages_processed",
		Help:      "A Counter that represents how many messages have passed through the given step in the pipeline",
	}, []string{
		"step_name",
		"step_type",
	})

	FilterDropped = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "clogger",
		Name:      "filter_dropped",
		Help:      "The number of messages dropped by the given filter",
	}, []string{
		"step_name",
	})

	OutputState = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "clogger",
		Name:      "output_state",
		Help:      "A Boolean that is 1 when the given output step is in the given state, and zero otherwise",
	}, []string{
		"step_name",
		"state",
	})
)

func InitMetrics(listenAddress string) {
	prometheus.MustRegister(MessagesProcessed, FilterDropped, OutputState)

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(listenAddress, nil)
}
