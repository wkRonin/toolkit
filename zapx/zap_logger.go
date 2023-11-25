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

package zapx

import (
	"go.uber.org/zap"
)

type ZapLogger struct {
	logger *zap.Logger
}

func NewZapLogger(l *zap.Logger) Logger {
	return &ZapLogger{
		logger: l,
	}
}

func (z *ZapLogger) Debug(msg string, args ...Field) {
	z.logger.Debug(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Info(msg string, args ...Field) {
	z.logger.Info(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Warn(msg string, args ...Field) {
	z.logger.Warn(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Error(msg string, args ...Field) {
	z.logger.Error(msg, z.toArgs(args)...)
}

func (z *ZapLogger) toArgs(args []Field) []zap.Field {
	res := make([]zap.Field, 0, len(args))
	for _, ar := range args {
		if ar.Key == "phone" {
			newAr := MaskPhone(ar.Key, ar.Value.(string))
			res = append(res, zap.Any(newAr.Key, newAr.String))
			continue
		}
		res = append(res, zap.Any(ar.Key, ar.Value))
	}
	return res
}
