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

package grpcx

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	uuid "github.com/satori/go.uuid"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/wkRonin/toolkit/logger"
	"github.com/wkRonin/toolkit/netx"
)

type Server struct {
	*grpc.Server
	Port int
	// 向前兼容，默认依旧使用etcd，要用consul时才把这个字段设置成true,并传递consul地址
	UseConsulClient bool
	EtcdAddrs       []string
	ConsulAddrs     string
	Name            string
	L               logger.Logger
	// 微服务是否以host模式运行，是host则自动获取本地ip地址注册到注册中心
	// 不是host模式则以Name作为微服务访问地址（k8s中则为svc名称，docker中则为container名称）
	IsHost bool
	// 是否使用本仓库的权重负载均衡，使用则传递Weight为权重值
	UseWrr bool
	// 使用权重负载均衡，又没传weight则默认为10
	// weight建议为10的整数倍
	Weight       int
	kaCancel     func()
	em           endpoints.Manager
	etcdClient   *etcdv3.Client
	consulClient *consulapi.Client
	key          string
}

// Serve 启动服务器并且阻塞
func (s *Server) Serve() error {
	l, err := net.Listen("tcp", ":"+strconv.Itoa(s.Port))
	if err != nil {
		return err
	}
	if s.UseConsulClient {
		// 服务注册到consul
		err = s.consulRegister()
	} else {
		// 服务注册到etcd
		err = s.etcdRegister()
	}
	if err != nil {
		return err
	}
	return s.Server.Serve(l)
}

func (s *Server) etcdRegister() error {
	client, err := etcdv3.New(etcdv3.Config{
		Endpoints: s.EtcdAddrs,
	})
	if err != nil {
		return err
	}
	s.etcdClient = client
	// endpoint 以服务为维度。一个服务一个 Manager
	em, err := endpoints.NewManager(client, "service/"+s.Name)
	if err != nil {
		return err
	}
	var addr string
	addr = s.getRegisterMessage()["addr"]
	key := "service/" + s.Name + "/" + addr
	s.key = key
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// 续租时间暂时不需要做成可配置，默认30秒即可
	var ttl int64 = 30
	leaseResp, err := client.Grant(ctx, ttl)

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ep := endpoints.Endpoint{Addr: addr}
	if s.UseWrr {
		ep.Metadata = map[string]any{
			"weight": s.Weight,
		}
	}
	err = em.AddEndpoint(ctx, key, ep, etcdv3.WithLease(leaseResp.ID))

	kaCtx, kaCancel := context.WithCancel(context.Background())
	s.kaCancel = kaCancel
	ch, err := client.KeepAlive(kaCtx, leaseResp.ID)
	if err != nil {
		return err
	}
	go func() {
		for kaResp := range ch {
			s.L.Debug("etcd续约", logger.String("response", kaResp.String()))
		}
	}()
	return nil
}

func (s *Server) consulRegister() error {
	cfg := consulapi.DefaultConfig()
	cfg.Address = s.ConsulAddrs
	var err error
	s.consulClient, err = consulapi.NewClient(cfg)

	if err != nil {
		return err
	}
	ipOrAddr := s.getRegisterMessage()
	check := &consulapi.AgentServiceCheck{
		GRPC:                           ipOrAddr["addr"],
		Timeout:                        "5s",
		Interval:                       "5s",
		DeregisterCriticalServiceAfter: "10s",
	}
	registration := &consulapi.AgentServiceRegistration{
		Name:    "service/" + s.Name,
		ID:      fmt.Sprintf("%s", uuid.NewV4()),
		Port:    s.Port,
		Tags:    []string{s.Name},
		Address: ipOrAddr["ip"],
		Check:   check,
	}
	if s.UseWrr {
		registration.Meta = map[string]string{
			"weight": strconv.Itoa(s.Weight)}
	}
	err = s.consulClient.Agent().ServiceRegister(registration)
	healthv1.RegisterHealthServer(s.Server, health.NewServer())
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) getRegisterMessage() map[string]string {
	res := make(map[string]string, 2)
	if s.IsHost {
		ip := netx.GetOutboundIP()
		res["addr"] = ip + ":" + strconv.Itoa(s.Port)
		res["ip"] = ip
		return res
	} else {
		res["addr"] = s.Name + ":" + strconv.Itoa(s.Port)
		res["ip"] = s.Name
		return res
	}
}
func (s *Server) Close() error {
	if s.kaCancel != nil {
		s.kaCancel()
	}
	if s.em != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err := s.em.DeleteEndpoint(ctx, s.key)
		if err != nil {
			return err
		}
	}
	if s.etcdClient != nil {
		err := s.etcdClient.Close()
		if err != nil {
			return err
		}
	}
	s.Server.GracefulStop()
	return nil
}
