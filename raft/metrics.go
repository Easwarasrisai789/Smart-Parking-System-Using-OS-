package raft

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry        *prometheus.Registry
	rpcRequests     *prometheus.CounterVec
	rpcRequestTimes *prometheus.HistogramVec
	nodeID          string
}

func NewMetrics(node *Raft) (*Metrics, error) {
	if node == nil {
		return nil, fmt.Errorf("Raft node is required")
	}

	nodeID := strconv.Itoa(node.ID)
	registry := prometheus.NewRegistry()

	metrics := &Metrics{
		registry: registry,
		rpcRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "raft_rpc_requests_total",
				Help: "Total Raft RPC requests handled by this node.",
			},
			[]string{"node_id", "method"},
		),
		rpcRequestTimes: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "raft_rpc_request_duration_seconds",
				Help:    "Duration of handled Raft RPC requests in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"node_id", "method"},
		),
		nodeID: nodeID,
	}

	collectors := []prometheus.Collector{
		metrics.rpcRequests,
		metrics.rpcRequestTimes,
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name:        "raft_current_term",
				Help:        "Current Raft term on this node.",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
			},
			func() float64 {
				return float64(node.GetCurrentTerm())
			},
		),
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name:        "raft_is_leader",
				Help:        "Whether this node currently believes it is the Raft leader.",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
			},
			func() float64 {
				if node.GetState() == Leader {
					return 1
				}
				return 0
			},
		),
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name:        "raft_commit_index",
				Help:        "Current committed log index on this node.",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
			},
			func() float64 {
				return float64(node.GetCommitIndex())
			},
		),
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name:        "raft_log_entries",
				Help:        "Number of log entries held by this node.",
				ConstLabels: prometheus.Labels{"node_id": nodeID},
			},
			func() float64 {
				lastIndex, _ := node.GetLogInfo()
				return float64(lastIndex + 1)
			},
		),
	}

	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			return nil, fmt.Errorf("register Raft metric: %w", err)
		}
	}

	return metrics, nil
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) observeRPC(method string) func() {
	if m == nil {
		return func() {}
	}

	labels := []string{m.nodeID, method}
	started := time.Now()
	m.rpcRequests.WithLabelValues(labels...).Inc()

	return func() {
		m.rpcRequestTimes.
			WithLabelValues(labels...).
			Observe(time.Since(started).Seconds())
	}
}
