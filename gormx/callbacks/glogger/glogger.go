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
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/utils"

	"github.com/wkRonin/toolkit/logger"
)

// GormLoggerCallbacks 利用callback实现的gorm plugin
// 用来替换gorm自带的logger
type GormLoggerCallbacks struct {
	// 慢sql阈值
	SlowThreshold time.Duration
	l             logger.Logger
}

func NewGormLoggerCallBacks(l logger.Logger, s time.Duration) *GormLoggerCallbacks {
	return &GormLoggerCallbacks{
		l:             l,
		SlowThreshold: s,
	}
}

func (c *GormLoggerCallbacks) before() func(db *gorm.DB) {
	return func(db *gorm.DB) {
		startTime := time.Now()
		db.Set("start_time", startTime)
	}
}

func (c *GormLoggerCallbacks) after() func(db *gorm.DB) {
	return func(db *gorm.DB) {
		val, _ := db.Get("start_time")
		start, ok := val.(time.Time)
		if !ok {
			return
		}
		duration := time.Since(start).Seconds() * 1000
		table := db.Statement.Table
		// 表名可能为空
		if table == "" {
			table = "unknown"
		}
		sql := db.Dialector.Explain(db.Statement.SQL.String(), db.Statement.Vars...)
		logEntry := &GormLog{
			Position:      utils.FileWithLineNum(),
			Duration:      duration,
			SQL:           sql,
			Rows:          db.Statement.RowsAffected,
			SlowThreshold: c.SlowThreshold,
			l:             c.l,
		}
		logEntry.String()
	}
}

func (c *GormLoggerCallbacks) Name() string {
	return "GormLogger"
}

func (c *GormLoggerCallbacks) Initialize(db *gorm.DB) error {
	return c.registerAll(db)
}

func (c *GormLoggerCallbacks) registerAll(db *gorm.DB) error {
	err := db.Callback().Query().Before("*").
		Register("gLogger_query_before", c.before())
	if err != nil {
		return err
	}

	err = db.Callback().Query().After("*").
		Register("gLogger_query_after", c.after())
	if err != nil {
		return err
	}

	err = db.Callback().Raw().Before("*").
		Register("gLogger_raw_before", c.before())
	if err != nil {
		return err
	}

	err = db.Callback().Raw().After("*").
		Register("gLogger_raw_after", c.after())
	if err != nil {
		return err
	}

	err = db.Callback().Row().Before("*").
		Register("gLogger_row_before", c.before())
	if err != nil {
		return err
	}

	err = db.Callback().Row().After("*").
		Register("gLogger_row_after", c.after())
	if err != nil {
		return err
	}

	err = db.Callback().Create().Before("*").
		Register("gLogger_create_before", c.before())
	if err != nil {
		return err
	}

	err = db.Callback().Create().After("*").
		Register("gLogger_create_after", c.after())
	if err != nil {
		return err
	}

	err = db.Callback().Update().Before("*").
		Register("gLogger_update_before", c.before())
	if err != nil {
		return err
	}

	err = db.Callback().Update().After("*").
		Register("gLogger_update_after", c.after())
	if err != nil {
		return err
	}

	err = db.Callback().Delete().Before("*").
		Register("gLogger_delete_before", c.before())
	if err != nil {
		return err
	}

	err = db.Callback().Delete().After("*").
		Register("gLogger_delete_after", c.after())
	if err != nil {
		return err
	}
	return nil
}
