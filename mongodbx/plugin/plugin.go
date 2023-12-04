package plugin

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/event"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/wkRonin/toolkit/logger"
)

type (
	Started   func(ctx context.Context, startedEvent *event.CommandStartedEvent)
	Succeeded func(ctx context.Context, succeededEvent *event.CommandSucceededEvent)
	Failed    func(ctx context.Context, failedEvent *event.CommandFailedEvent)
)

type MongoPluginMonitor struct {
	vector *prometheus.SummaryVec
	tracer trace.Tracer
	l      logger.Logger
	ctx    context.Context
}

func NewMongoPluginMonitor(opt prometheus.SummaryOpts, l logger.Logger) *MongoPluginMonitor {
	vector := prometheus.NewSummaryVec(opt, []string{"cmd"})
	prometheus.MustRegister(vector)
	return &MongoPluginMonitor{
		vector: vector,
		tracer: otel.GetTracerProvider().Tracer("github.com/wkRonin/toolkit/mongodbx/plugin"),
		l:      l,
	}
}

func (p *MongoPluginMonitor) StartedProm() Started {
	return func(ctx context.Context, startedEvent *event.CommandStartedEvent) {
		var span trace.Span
		p.ctx, span = p.tracer.Start(ctx, "mongodbx: "+startedEvent.CommandName, trace.WithSpanKind(trace.SpanKindClient))
		span.SetAttributes(attribute.String("mongo.database", startedEvent.DatabaseName))
		span.SetAttributes(attribute.String("mongo.command", startedEvent.Command.String()))
		span.SetAttributes(attribute.String("mongo.command.name", startedEvent.CommandName))
		p.l.Debug("mongo", logger.Any("mongoCommand", startedEvent.Command))
	}
}
func (p *MongoPluginMonitor) SucceededProm() Succeeded {
	return func(ctx context.Context, succeededEvent *event.CommandSucceededEvent) {
		duration := succeededEvent.Duration
		cmd := succeededEvent.CommandName
		p.vector.WithLabelValues(cmd).
			Observe(float64(duration.Milliseconds()))

		span := trace.SpanFromContext(p.ctx)
		if !span.IsRecording() {
			return
		}
		defer span.End()
	}
}

func (p *MongoPluginMonitor) FailedProm() Failed {
	return func(ctx context.Context, failedEvent *event.CommandFailedEvent) {
		duration := failedEvent.Duration
		cmd := failedEvent.CommandName
		p.vector.WithLabelValues(cmd).
			Observe(float64(duration.Milliseconds()))

		span := trace.SpanFromContext(p.ctx)
		if !span.IsRecording() {
			return
		}
		defer span.End()
		span.SetStatus(codes.Error, failedEvent.Failure)
	}
}
