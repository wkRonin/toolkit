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

package ratelimit

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wkRonin/toolkit/logger"
	"github.com/wkRonin/toolkit/ratelimit"
)

type MiddlewareBuilder struct {
	prefix  string
	limiter ratelimit.Limiter
	l       logger.Logger
}

func NewMiddlewareBuilder(limiter ratelimit.Limiter, l logger.Logger) *MiddlewareBuilder {
	return &MiddlewareBuilder{
		prefix:  "ip-limiter",
		limiter: limiter,
		l:       l,
	}
}

func (b *MiddlewareBuilder) Prefix(prefix string) *MiddlewareBuilder {
	b.prefix = prefix
	return b
}

func (b *MiddlewareBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		limited, err := b.limit(ctx)
		if err != nil {
			b.l.Error("err from limit", logger.Error(err))
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if limited {
			b.l.Warn("has been limited", logger.String("ip", ctx.ClientIP()))
			ctx.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		ctx.Next()
	}
}

func (b *MiddlewareBuilder) limit(ctx *gin.Context) (bool, error) {
	key := fmt.Sprintf("%s:%s", b.prefix, ctx.ClientIP())
	return b.limiter.Limit(ctx.Request.Context(), key)
}
