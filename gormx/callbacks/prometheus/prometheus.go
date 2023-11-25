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

package prometheus

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type Callbacks struct {
	vector *prometheus.SummaryVec
}

// Register 使用gorm的callback 采集增删改查的sql响应时间
func (c *Callbacks) Register(opt prometheus.SummaryOpts, db *gorm.DB) error {
	vector := prometheus.NewSummaryVec(opt, []string{"type", "table"})
	prometheus.MustRegister(vector)
	c.vector = vector
	return c.registerAll(db)
}

func (c *Callbacks) registerAll(db *gorm.DB) error {
	err := db.Callback().Query().Before("*").
		Register("prometheus_query_before", c.before("query"))
	if err != nil {
		return err
	}

	err = db.Callback().Query().After("*").
		Register("prometheus_query_after", c.after("query"))
	if err != nil {
		return err
	}

	err = db.Callback().Raw().Before("*").
		Register("prometheus_raw_before", c.before("raw"))
	if err != nil {
		return err
	}

	err = db.Callback().Raw().After("*").
		Register("prometheus_raw_after", c.after("raw"))
	if err != nil {
		return err
	}

	err = db.Callback().Row().Before("*").
		Register("prometheus_row_before", c.before("row"))
	if err != nil {
		return err
	}

	err = db.Callback().Row().After("*").
		Register("prometheus_row_after", c.after("row"))
	if err != nil {
		return err
	}

	err = db.Callback().Create().Before("*").
		Register("prometheus_create_before", c.before("create"))
	if err != nil {
		return err
	}

	err = db.Callback().Create().After("*").
		Register("prometheus_create_after", c.after("create"))
	if err != nil {
		return err
	}

	err = db.Callback().Update().Before("*").
		Register("prometheus_update_before", c.before("update"))
	if err != nil {
		return err
	}

	err = db.Callback().Update().After("*").
		Register("prometheus_update_after", c.after("update"))
	if err != nil {
		return err
	}

	err = db.Callback().Delete().Before("*").
		Register("prometheus_delete_before", c.before("delete"))
	if err != nil {
		return err
	}

	err = db.Callback().Delete().After("*").
		Register("prometheus_delete_after", c.after("delete"))
	if err != nil {
		return err
	}
	return nil
}

func (c *Callbacks) before(typ string) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		start := time.Now()
		db.Set("start_time", start)
	}
}

func (c *Callbacks) after(typ string) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		val, _ := db.Get("start_time")
		// 如果上面没找到，这边必然断言失败
		start, ok := val.(time.Time)
		if !ok {
			return
		}
		duration := time.Since(start)
		table := db.Statement.Table
		// 表名可能为空
		if table == "" {
			table = "unknown"
		}
		c.vector.WithLabelValues(typ, table).
			Observe(float64(duration.Milliseconds()))
	}
}
