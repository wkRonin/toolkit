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

package log

import (
	"context"
	"fmt"
	"runtime"
	"time"

	jsoniter "github.com/json-iterator/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/wkRonin/toolkit/grpcx/interceptors"
	"github.com/wkRonin/toolkit/logger"
	"github.com/wkRonin/toolkit/netx"
)

type LoggerInterceptorBuilder struct {
	l logger.Logger
	interceptors.Builder
	// 自己的名字
	serviceName string
	// 远端的名字
	peerName string
}

func NewLoggerInterceptorBuilder(l logger.Logger, peerName string) *LoggerInterceptorBuilder {
	return &LoggerInterceptorBuilder{
		l:        l,
		peerName: peerName,
	}
}

func (b *LoggerInterceptorBuilder) BuildUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		// 默认过滤掉该探活日志
		if info.FullMethod == "/grpc.health.v1.Health/Check" {
			return handler(ctx, req)
		}

		var start = time.Now()
		var fields = make([]logger.Field, 0, 20)
		var event = "normal"

		defer func() {
			cost := time.Since(start)
			if rec := recover(); rec != nil {
				switch recType := rec.(type) {
				case error:
					err = recType
				default:
					err = fmt.Errorf("%v", rec)
				}
				stack := make([]byte, 4096)
				stack = stack[:runtime.Stack(stack, true)]
				event = "recover"
				err = status.New(codes.Internal, "panic, err "+err.Error()).Err()
			}
			statusInfo, _ := status.FromError(err)
			fields = append(fields,
				logger.String("type", "UnaryServer"),
				logger.String("code", statusInfo.Code().String()),
				logger.String("code_msg", statusInfo.Message()),
				logger.String("event", event),
				logger.String("method", info.FullMethod),
				logger.String("cost", cost.String()),
				logger.String("peer", b.PeerName(ctx)),
				logger.String("peer_ip", b.PeerIP(ctx)),
			)

			var reqMap = map[string]interface{}{
				"payload": b.stringToJSON(req),
			}
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				reqMap["metadata"] = md
			}
			fields = append(fields,
				logger.Any("req", reqMap),
				logger.Any("res", map[string]interface{}{
					"payload": b.stringToJSON(resp),
				}))
			if err != nil {
				fields = append(fields, logger.Error(err))
				b.l.Error("grpc.response", fields...)
			} else {
				b.l.Info("grpc.response", fields...)
			}

		}()

		return handler(ctx, req)
	}
}

func (b *LoggerInterceptorBuilder) BuildUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		beg := time.Now()
		// 获取对端信息
		var p peer.Peer
		// 响应的头信息
		var resHeader metadata.MD
		// 响应的尾信息
		var resTrailer metadata.MD
		// 请求的头信息
		reqHeader, _ := metadata.FromOutgoingContext(ctx)
		opts = append(opts, grpc.Header(&resHeader))
		opts = append(opts, grpc.Trailer(&resTrailer))
		opts = append(opts, grpc.Peer(&p))
		err := invoker(ctx, method, req, reply, cc, opts...)
		statusInfo, _ := status.FromError(err)
		// 请求
		var reqMap = map[string]any{
			"payload":  b.stringToJSON(req),
			"metadata": reqHeader,
		}
		var resMap = map[string]any{
			"payload": b.stringToJSON(reply),
			"metadata": map[string]any{
				"header":  resHeader,
				"trailer": resTrailer,
			},
		}
		// 记录此次调用grpc的耗时
		cost := time.Since(beg)
		var fields = make([]logger.Field, 0, 9)
		fields = append(fields,
			logger.Any("req", reqMap),
			logger.String("type", "UnaryClient"),
			logger.String("code", statusInfo.Code().String()),
			logger.String("code_msg", statusInfo.Message()),
			logger.String("method", method),
			logger.String("cost", cost.String()),
			logger.String("peer", b.peerName),
			logger.String("peer_ip", p.Addr.String()),
		)
		if err != nil {
			fields = append(fields, logger.Error(err))
			b.l.Error("grpc.response", fields...)
		} else {
			fields = append(fields, logger.Any("res", resMap))
			b.l.Info("grpc.response", fields...)
		}
		ctx = b.setClientMetadata(ctx)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func (b *LoggerInterceptorBuilder) setClientMetadata(ctx context.Context) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	// 判断一下 防止和其它interceptor重复设置
	if md.Get(interceptors.PeerIPKey) == nil {
		// 设置自己的ip
		md.Set(interceptors.PeerIPKey, netx.GetOutboundIP())
	}
	if md.Get(interceptors.PeerNameKey) == nil {
		// 设置自己的Name
		md.Set(interceptors.PeerNameKey, b.serviceName)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

func (b *LoggerInterceptorBuilder) stringToJSON(obj interface{}) string {
	var jsonAPI = jsoniter.Config{
		SortMapKeys:            true,
		UseNumber:              true,
		CaseSensitive:          true,
		EscapeHTML:             true,
		ValidateJsonRawMessage: true,
	}.Froze()
	aa, _ := jsonAPI.Marshal(obj)
	return string(aa)
}
