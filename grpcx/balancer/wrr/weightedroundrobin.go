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

package wrr

import (
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/grpclog"
)

const name = "custom_wrr"

// balancer.Balancer 接口
// balancer.Builder 接口
// balancer.Picker 接口
// base.PickerBuilder 接口
// Balancer 是 Picker 的装饰器
func init() {
	balancer.Register(newBuilder())
}

func newBuilder() balancer.Builder {
	// NewBalancerBuilder 是帮我们把一个 Picker Builder 转化为一个 balancer.Builder
	return base.NewBalancerBuilder(name,
		&PickerBuilder{}, base.Config{HealthCheck: false})
}

// PickerBuilder 传统版本的基于权重的负载均衡算法
type PickerBuilder struct {
}

func (p *PickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	grpclog.Infof("%s_picker: newPicker called with info: %v", name, info)
	lenR := len(info.ReadySCs)
	if lenR == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}
	conns := make([]*conn, 0, lenR)
	// sc => SubConn
	// sci => SubConnInfo
	for sc, sci := range info.ReadySCs {
		cc := &conn{
			cc: sc,
		}
		// Metadata已经被废弃
		// endpoints.Endpoint中只有Metadata无法使用attributes传递权重信息
		// 所以这里获取权重就只能从Metadata中获取
		md, ok := sci.Address.Metadata.(map[string]any)
		if ok {
			weightVal := md["weight"]
			weight, _ := weightVal.(float64)
			cc.weight = int(weight)
		}
		if cc.weight == 0 {
			// 给个默认值
			cc.weight = 10
		}
		cc.currentWeight = cc.weight
		conns = append(conns, cc)
	}
	return &Picker{
		conns: conns,
	}
}

type Picker struct {
	conns []*conn
	mutex sync.Mutex
}

// Pick 基于权重的负载均衡算法
func (p *Picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if len(p.conns) == 0 {
		// 没有候选节点
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	var total int
	var maxCC *conn
	// 计算当前权重
	for _, cc := range p.conns {
		// 性能最好就是在 cc 上用原子操作
		// 但是筛选结果不会严格符合 WRR 算法
		total += cc.weight
		cc.currentWeight += cc.weight
		if maxCC == nil || cc.currentWeight > maxCC.currentWeight {
			maxCC = cc
		}
	}

	// 更新
	maxCC.currentWeight -= total
	return balancer.PickResult{
		SubConn: maxCC.cc,
		// Done: func(info balancer.DoneInfo) {
		// 根据调用结果来调整权重
		// },
	}, nil
}

// conn 代表节点
type conn struct {
	// 权重
	weight        int
	currentWeight int
	cc            balancer.SubConn
}
