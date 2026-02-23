package scheduler

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	TaskExecutionsTotal   *prometheus.CounterVec
	TaskExecutionDuration *prometheus.HistogramVec
	TaskExecutionSuccess  *prometheus.CounterVec
	TaskExecutionFailures *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		TaskExecutionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "task_scheduler_executions_total",
				Help: "Total number of task executions",
			},
			[]string{"task_id"},
		),
		TaskExecutionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "task_scheduler_execution_duration_ms",
				Help:    "Task execution duration in milliseconds",
				Buckets: prometheus.ExponentialBuckets(10, 2, 10),
			},
			[]string{"task_id"},
		),
		TaskExecutionSuccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "task_scheduler_execution_success_total",
				Help: "Total number of successful task executions",
			},
			[]string{"task_id"},
		),
		TaskExecutionFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "task_scheduler_execution_failures_total",
				Help: "Total number of failed task executions",
			},
			[]string{"task_id"},
		),
	}
}

func (m *Metrics) RecordTaskExecution(taskID string, success bool, durationMs int64) {
	m.TaskExecutionsTotal.WithLabelValues(taskID).Inc()
	m.TaskExecutionDuration.WithLabelValues(taskID).Observe(float64(durationMs))

	if success {
		m.TaskExecutionSuccess.WithLabelValues(taskID).Inc()
	} else {
		m.TaskExecutionFailures.WithLabelValues(taskID).Inc()
	}
}
