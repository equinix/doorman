package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ActiveClientTotal               prometheus.Gauge
	AuthenticationDuration          prometheus.Histogram
	AuthenticationFailureTotalCount prometheus.Counter
	AuthenticationSuccessTotalCount prometheus.Counter
	ErrorTotal                      *prometheus.CounterVec
)

func Init() {
	initActiveClientTotalCounter()
	initAuthenticationDuration()
	initAuthenticationFailureTotalCount()
	initAuthenticationSuccessTotalCount()
	initErrorTotalCounter()

	prometheus.MustRegister(ActiveClientTotal)
	prometheus.MustRegister(AuthenticationDuration)
	prometheus.MustRegister(AuthenticationFailureTotalCount)
	prometheus.MustRegister(AuthenticationSuccessTotalCount)
	prometheus.MustRegister(ErrorTotal)

}

func initActiveClientTotalCounter() {
	ActiveClientTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "active_clients",
		Subsystem: "doorman",
		Help:      "Number of active clients.",
	})
}

func initAuthenticationDuration() {
	buckets := []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

	AuthenticationDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "authentication_duration",
		Help:    "Histogram of authentication time for handler in seconds.",
		Buckets: buckets,
	})
}

func initAuthenticationFailureTotalCount() {
	AuthenticationFailureTotalCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "authentication_failures",
		Help: "Number of total failed authentication attempts.",
	})
}

func initAuthenticationSuccessTotalCount() {
	AuthenticationSuccessTotalCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "authentication_success",
		Help: "Number of total success authentication attempts.",
	})
}

func initErrorTotalCounter() {
	ErrorTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "number_of_errors",
		Subsystem: "doorman",
		Help:      "Number of total errors.",
	}, []string{"op", "type"})

	labelValues := []prometheus.Labels{
		{"op": "doorman", "type": "errors"},
	}

	initCounterLabels(ErrorTotal, labelValues)
}

func initCounterLabels(m *prometheus.CounterVec, l []prometheus.Labels) {
	for _, labels := range l {
		m.With(labels)
	}
}
