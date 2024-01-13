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

package interceptors

import (
	"context"
	"net"
	"strings"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	PeerNameKey = "app"
	PeerIPKey   = "client-ip"
)

type Builder struct {
}

// PeerName 获取对端应用名称
func (b *Builder) PeerName(ctx context.Context) string {
	return b.getGrpcHeaderValue(ctx, PeerNameKey)
}

// PeerIP 获取对端ip
func (b *Builder) PeerIP(ctx context.Context) string {
	// 如果在 ctx 里面传入。或者说客户端里面设置了，就直接用它设置的
	// 如果是经过网关的客户端，就需要客户端主动设置，防止后面拿到网关的 IP
	clientIP := b.getGrpcHeaderValue(ctx, PeerIPKey)
	if clientIP != "" {
		return clientIP
	}

	// 从grpc里取对端ip
	pr, ok2 := peer.FromContext(ctx)
	if !ok2 {
		return ""
	}
	if pr.Addr == net.Addr(nil) {
		return ""
	}
	addSlice := strings.Split(pr.Addr.String(), ":")
	if len(addSlice) > 1 {
		return addSlice[0]
	}
	return ""
}

func (b *Builder) getGrpcHeaderValue(ctx context.Context, key string) string {
	if key == "" {
		return ""
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return strings.Join(md.Get(key), ";")
}
