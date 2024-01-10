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

package glogger

import (
	"fmt"
	"time"

	"github.com/wkRonin/toolkit/logger"
)

type GormLog struct {
	Position      string
	Duration      float64
	SQL           string
	Rows          int64
	SlowThreshold time.Duration
	l             logger.Logger
}

func (gl GormLog) String() {
	// 计算出多少毫秒
	thresholdMillis := float64(gl.SlowThreshold.Nanoseconds()) / float64(time.Millisecond)
	var logStr string
	if gl.Rows == -1 {
		logStr = fmt.Sprintf("Position: %s | Duration: %.4fms | SQL: %s | Rows: -", gl.Position, gl.Duration, gl.SQL)
	} else {
		logStr = fmt.Sprintf("Position: %s | Duration: %.4fms | SQL: %s | Rows: %d", gl.Position, gl.Duration, gl.SQL, gl.Rows)
	}
	if thresholdMillis <= gl.Duration {
		logStr = fmt.Sprintf("%s | Is Slow Query SQL", logStr)
	}
	gl.l.Info("Gorm", logger.String("GormInfo", logStr))
}
