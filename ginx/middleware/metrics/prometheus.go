/*
 *    Copyright 2023 wkRonin
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

type MiddlewareBuilder struct {
	Namespace  string
	Subsystem  string
	Name       string
	Help       string
	InstanceID string
}

// Build gin的http响应时间和当前活跃请求数
func (m *MiddlewareBuilder) Build() gin.HandlerFunc {
	labels := []string{"method", "pattern", "status"}
	summary := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: m.Namespace,
		Subsystem: m.Subsystem,
		Name:      m.Name + "_resp_time",
		Help:      m.Help,
		ConstLabels: map[string]string{
			"instance_id": m.InstanceID,
		},
		Objectives: map[float64]float64{
			0.5:   0.01,
			0.75:  0.01,
			0.90:  0.01,
			0.99:  0.001,
			0.999: 0.0001,
		},
	}, labels)
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		// 当前活跃请求数
		Namespace: m.Namespace,
		Subsystem: m.Subsystem,
		Name:      m.Name + "_active_req",
		Help:      m.Help,
		ConstLabels: map[string]string{
			"instance_id": m.InstanceID,
		},
	})
	prometheus.MustRegister(summary, gauge)
	return func(ctx *gin.Context) {
		start := time.Now()
		gauge.Inc()
		defer func() {
			duration := time.Since(start)
			gauge.Desc()
			method := ctx.Request.Method
			pattern := ctx.FullPath()
			if pattern == "" {
				pattern = "unknown"
			}
			summary.WithLabelValues(
				method,
				pattern,
				strconv.Itoa(ctx.Writer.Status())).
				Observe(float64(duration.Milliseconds()))
		}()
		ctx.Next()
	}
}
