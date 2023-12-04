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

package ginx

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/wkRonin/toolkit/logger"
)

// 暂无更好的办法，先用包变量
var vector *prometheus.CounterVec

// InitCounterCode 初始化错误码统计：prometheus错误码统计
func InitCounterCode(opt prometheus.CounterOpts) {
	vector = prometheus.NewCounterVec(opt, []string{"code"})
	prometheus.MustRegister(vector)
}

/*
Wrap系列函数说明：
1、现在只能处理请求体（泛型支持），url参数暂不支持
2、ctx中的取出来的值限制实现了jwt.Claims的接口
*/

// WrapReq 统一处理请求体bind/错误日志打印
func WrapReq[T any](fn func(ctx *gin.Context, req T) (Result, error),
	l logger.Logger,
	lm LogMessage) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req T
		if err := ctx.Bind(&req); err != nil {
			l.Error("请求参数错误",
				logger.String("method", lm.Method),
				logger.Error(err))
			return
		}
		res, err := fn(ctx, req)
		vector.WithLabelValues(strconv.Itoa(res.Code)).Inc()
		if err != nil {
			l.Error(lm.Message,
				logger.String("method", lm.Method),
				logger.Error(err),
				// 命中的路由
				logger.String("route", ctx.FullPath()))
		}
		ctx.JSON(http.StatusOK, res)
	}
}

// WrapReqAndToken 统一处理请求体bind/ctx中取值/错误日志打印
func WrapReqAndToken[T any, C jwt.Claims](fn func(ctx *gin.Context, req T, uc C) (Result, error),
	l logger.Logger,
	lm LogMessage,
	ctxKey string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req T
		if err := ctx.Bind(&req); err != nil {
			l.Error("请求参数错误",
				logger.String("method", lm.Method),
				logger.Error(err))
			return
		}
		uc, ok := ctx.Get(ctxKey)
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			l.Warn("jwt中不存在用户信息",
				logger.String("method", lm.Method),
			)
			return
		}
		var c C
		c, ok = uc.(C)
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			l.Warn("jwt中用户信息非法",
				logger.String("method", lm.Method),
			)
			return
		}
		res, err := fn(ctx, req, c)
		vector.WithLabelValues(strconv.Itoa(res.Code)).Inc()
		if err != nil {
			l.Error(lm.Message,
				logger.String("method", lm.Method),
				logger.Error(err),
				// 命中的路由
				logger.String("route", ctx.FullPath()),
			)
		}
		ctx.JSON(http.StatusOK, res)
	}
}

// WrapToken 统一处理ctx中取值/错误日志打印
func WrapToken[C jwt.Claims](
	fn func(ctx *gin.Context, uc C) (Result, error),
	l logger.Logger,
	lm LogMessage,
	ctxKey string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		uc, ok := ctx.Get(ctxKey)
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			l.Warn("jwt中不存在用户信息",
				logger.String("method", lm.Method),
			)
			return
		}
		var c C
		c, ok = uc.(C)
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			l.Warn("jwt中用户信息非法",
				logger.String("method", lm.Method),
			)
			return
		}
		res, err := fn(ctx, c)
		vector.WithLabelValues(strconv.Itoa(res.Code)).Inc()
		if err != nil {
			l.Error(lm.Message,
				logger.String("method", lm.Method),
				logger.Error(err),
				// 命中的路由
				logger.String("route", ctx.FullPath()),
			)
		}
		ctx.JSON(http.StatusOK, res)
	}
}

// WrapError 统一处理错误日志打印
func WrapError(fn func(ctx *gin.Context) (Result, error),
	l logger.Logger,
	lm LogMessage) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		res, err := fn(ctx)
		vector.WithLabelValues(strconv.Itoa(res.Code)).Inc()
		if err != nil {
			l.Error(lm.Message,
				logger.String("method", lm.Method),
				// 命中的路由
				logger.String("route", ctx.FullPath()),
				logger.Error(err))
		}
		// 约定msg不为空才返回响应体
		if res.Msg != "" {
			ctx.JSON(http.StatusOK, res)
		}
	}
}

type Result struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

type LogMessage struct {
	Method  string
	Message string
}
