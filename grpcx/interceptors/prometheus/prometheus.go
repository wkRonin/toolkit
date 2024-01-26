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

package prometheus

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/wkRonin/toolkit/grpcx/interceptors"
)

type MetricInterceptorBuilder struct {
	Namespace  string
	Subsystem  string
	Name       string
	Help       string
	InstanceID string
	interceptors.Builder
}

func (b *MetricInterceptorBuilder) BuildUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	labels := []string{"type", "rpc_service", "method", "peer", "code"}
	summary := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: b.Namespace,
			Subsystem: b.Subsystem,
			Name:      b.Name + "server_handle_seconds",
			Objectives: map[float64]float64{
				0.5:   0.01,
				0.9:   0.01,
				0.95:  0.01,
				0.99:  0.001,
				0.999: 0.0001,
			},
			Help: b.Help,
			ConstLabels: map[string]string{
				"instance_id": b.InstanceID,
			},
		}, labels)
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		// 被请求总数
		Namespace: b.Namespace,
		Subsystem: b.Subsystem,
		Name:      b.Name + "server_handle_total",
		Help:      b.Help,
		ConstLabels: map[string]string{
			"instance_id": b.InstanceID,
		},
	}, labels)
	prometheus.MustRegister(summary, counter)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		start := time.Now()
		defer func() {
			serviceName, method := b.splitMethodName(info.FullMethod)
			duration := float64(time.Since(start).Milliseconds())
			pr := b.PeerName(ctx)
			st, _ := status.FromError(err)
			summary.WithLabelValues("unary", serviceName, method, pr, st.Code().String()).Observe(duration)
			counter.WithLabelValues("unary", serviceName, method, pr, st.Code().String()).Inc()

		}()
		resp, err = handler(ctx, req)
		return
	}
}

func (b *MetricInterceptorBuilder) BuildUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	labels := []string{"type", "name", "method", "peer", "code"}
	summary := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: b.Namespace,
			Subsystem: b.Subsystem,
			Name:      b.Name + "client_handle_seconds",
			Objectives: map[float64]float64{
				0.5:   0.01,
				0.9:   0.01,
				0.95:  0.01,
				0.99:  0.001,
				0.999: 0.0001,
			},
			Help: b.Help,
			ConstLabels: map[string]string{
				"instance_id": b.InstanceID,
			},
		}, labels)
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		// 被请求总数
		Namespace: b.Namespace,
		Subsystem: b.Subsystem,
		Name:      b.Name + "client_handle_total",
		Help:      b.Help,
		ConstLabels: map[string]string{
			"instance_id": b.InstanceID,
		},
	}, labels)
	prometheus.MustRegister(summary, counter)
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		start := time.Now()
		defer func() {
			duration := float64(time.Since(start).Milliseconds())
			st, _ := status.FromError(err)
			summary.WithLabelValues("unary", "grpc_client", method, cc.Target(), st.Code().String()).Observe(duration)
			counter.WithLabelValues("unary", "grpc_client", method, cc.Target(), st.Code().String()).Inc()
		}()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
func (b *MetricInterceptorBuilder) splitMethodName(fullMethodName string) (string, string) {
	fullMethodName = strings.TrimPrefix(fullMethodName, "/") // remove leading slash
	if i := strings.Index(fullMethodName, "/"); i >= 0 {
		return fullMethodName[:i], fullMethodName[i+1:]
	}
	return "unknown", "unknown"
}
