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

package trace

import (
	"context"
	"net"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type TracingHook struct {
	tracer    trace.Tracer
	isCluster bool
}

// NewTracingHook 使用redis的hook接口来提供trace链路追踪
func NewTracingHook(tracer string, isCluster bool) *TracingHook {
	return &TracingHook{
		tracer:    otel.GetTracerProvider().Tracer(tracer),
		isCluster: isCluster,
	}
}

func (t TracingHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (t TracingHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		ctx, span := t.tracer.Start(ctx, "redisx: "+cmd.Name(), trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()
		var cmdName string
		if t.isCluster {
			cmdName = cmd.FullName()
		} else {
			cmdName = cmd.Name()
		}
		span.SetAttributes(attribute.String("cmd.Name", cmdName))
		span.SetAttributes(attribute.String("cmd.String", cmd.String()))
		err := next(ctx, cmd)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
}

func (t TracingHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		return next(ctx, cmds)
	}
}
