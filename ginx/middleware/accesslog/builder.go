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

package accesslog

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/atomic"
)

type MiddlewareBuilder struct {
	logFunc       func(ctx context.Context, al AccessLog)
	allowReqBody  *atomic.Bool
	allowRespBody bool
}

func NewMiddlewareBuilder(fn func(ctx context.Context, al AccessLog)) *MiddlewareBuilder {
	return &MiddlewareBuilder{
		logFunc: fn,
		// 默认不打印
		allowReqBody: atomic.NewBool(false),
	}
}

func (b *MiddlewareBuilder) AllowReqBody(ok bool) *MiddlewareBuilder {
	b.allowReqBody.Store(ok)
	return b
}

func (b *MiddlewareBuilder) AllowRespBody() *MiddlewareBuilder {
	b.allowRespBody = true
	return b
}

func (b *MiddlewareBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()

		al := AccessLog{
			Method: ctx.Request.Method,
			Path:   ctx.Request.URL.Path,
		}
		if b.allowReqBody.Load() && ctx.Request.Body != nil {
			// 直接忽略 error，不影响程序运行
			reqBodyBytes, _ := ctx.GetRawData()
			// Request.Body 是一个 Stream（流）对象，所以只能读取一次
			// 因此读完之后要放回去，不然后续步骤是读不到的
			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
			al.ReqBody = string(reqBodyBytes)
		}

		if b.allowRespBody {
			ctx.Writer = responseWriter{
				ResponseWriter: ctx.Writer,
				al:             &al,
			}
		}

		defer func() {
			duration := time.Since(start)
			al.Duration = duration.String()
			b.logFunc(ctx, al)
		}()
		// 这里可以写业务代码
		ctx.Next()
	}
}

// AccessLog 自定义打印信息
type AccessLog struct {
	Method     string `json:"method"`
	Path       string `json:"path"`
	ReqBody    string `json:"req_body"`
	Duration   string `json:"duration"`
	StatusCode int    `json:"status_code"`
	RespBody   string `json:"resp_body"`
}

type responseWriter struct {
	al *AccessLog
	gin.ResponseWriter
}

func (r responseWriter) WriteHeader(statusCode int) {
	r.al.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r responseWriter) Write(data []byte) (int, error) {
	r.al.RespBody = string(data)
	return r.ResponseWriter.Write(data)
}

func (r responseWriter) WriteString(data string) (int, error) {
	r.al.RespBody = data
	return r.ResponseWriter.WriteString(data)
}
