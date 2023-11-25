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
	"context"
	"net"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

type PrometheusHook struct {
	vector    *prometheus.SummaryVec
	isCluster bool
}

// NewPrometheusHook 使用redis的hook接口采集cmd响应时间
func NewPrometheusHook(opt prometheus.SummaryOpts, isCluster bool) *PrometheusHook {
	vector := prometheus.NewSummaryVec(opt, []string{"cmd", "key_exist"})
	prometheus.MustRegister(vector)
	return &PrometheusHook{
		vector:    vector,
		isCluster: isCluster,
	}
}

func (p *PrometheusHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (p *PrometheusHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		var err error
		defer func() {
			duration := time.Since(start)
			keyExist := err == redis.Nil
			var cmdName string
			if p.isCluster {
				cmdName = cmd.FullName()
			} else {
				cmdName = cmd.Name()
			}
			p.vector.WithLabelValues(cmdName,
				strconv.FormatBool(keyExist)).
				Observe(float64(duration.Milliseconds()))
		}()
		err = next(ctx, cmd)
		return err
	}
}

func (p *PrometheusHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		return next(ctx, cmds)
	}
}
