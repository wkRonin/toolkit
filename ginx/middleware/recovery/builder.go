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

package recovery

import (
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/wkRonin/toolkit/logger"
)

type MiddlewareBuilder struct {
	allowWriteStack bool
	isAbort         bool
	l               logger.Logger
}

func NewMiddlewareBuilder(l logger.Logger) *MiddlewareBuilder {
	return &MiddlewareBuilder{
		allowWriteStack: false,
		isAbort:         false,
		l:               l,
	}
}

func (b *MiddlewareBuilder) AllowWriteStack() *MiddlewareBuilder {
	b.allowWriteStack = true
	return b
}

func (b *MiddlewareBuilder) IsAbort() *MiddlewareBuilder {
	b.isAbort = true
	return b
}

// Build 用于替换gin框架的Recovery中间件，因为传入参数，再包一层
func (b *MiddlewareBuilder) Build() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			// defer 延迟调用，出了异常，处理并恢复异常，记录日志
			if err := recover(); err != nil {
				//  这个不必须，检查是否存在断开的连接(broken pipe或者connection reset by peer)---------开始--------
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					var se *os.SyscallError
					if errors.As(ne, &se) {
						seStr := strings.ToLower(se.Error())
						if strings.Contains(seStr, "broken pipe") ||
							strings.Contains(seStr, "connection reset by peer") {
							brokenPipe = true
						}
					}
				}
				// httputil包预先准备好的DumpRequest方法
				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					b.l.Error(c.Request.URL.Path,
						logger.Any("error", err),
						logger.String("request", string(httpRequest)),
					)
					// 如果连接已断开，我们无法向其写入状态
					c.Error(err.(error))
					c.Abort()
					return
				}
				//  这个不必须，检查是否存在断开的连接(broken pipe或者connection reset by peer)---------结束--------

				// 是否打印堆栈信息，使用的是debug.Stack()，传入false，在日志中就没有堆栈信息
				if b.allowWriteStack {
					b.l.Error("[Recovery from panic]",
						logger.Any("error", err),
						logger.String("request", string(httpRequest)),
						logger.String("stack", string(debug.Stack())),
					)
				} else {
					b.l.Error("[Recovery from panic]",
						logger.Any("error", err),
						logger.String("request", string(httpRequest)),
					)
				}
				if b.isAbort {
					// 返回500状态码
					c.AbortWithStatus(http.StatusInternalServerError)
				} else {
					// 该方式前端不报错
					c.String(200, "系统错误")
				}
			}
		}()
		c.Next()
	}
}
