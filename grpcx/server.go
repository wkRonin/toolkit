package grpcx

import (
	"context"
	"net"
	"strconv"
	"time"

	etcdv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"google.golang.org/grpc"

	"github.com/wkRonin/toolkit/logger"
	"github.com/wkRonin/toolkit/netx"
)

type Server struct {
	*grpc.Server
	Port      int
	EtcdAddrs []string
	Name      string
	L         logger.Logger
	// 微服务是否以host模式运行，是host则自动获取本地ip地址注册到注册中心
	// 不是host模式则以Name作为微服务访问地址（k8s中则为svc名称，docker中则为container名称）
	IsHost bool
	// 是否使用本仓库的权重负载均衡，使用则传递Weight为权重值
	UseWrr bool
	// 使用权重负载均衡，又没传weight则默认为10
	// weight建议为10的整数倍
	Weight   int
	kaCancel func()
	em       endpoints.Manager
	client   *etcdv3.Client
	key      string
}

// Serve 启动服务器并且阻塞
func (s *Server) Serve() error {
	l, err := net.Listen("tcp", ":"+strconv.Itoa(s.Port))
	if err != nil {
		return err
	}
	// 服务注册到etcd
	err = s.register()
	if err != nil {
		return err
	}
	return s.Server.Serve(l)
}

func (s *Server) register() error {
	client, err := etcdv3.New(etcdv3.Config{
		Endpoints: s.EtcdAddrs,
	})
	if err != nil {
		return err
	}
	s.client = client
	// endpoint 以服务为维度。一个服务一个 Manager
	em, err := endpoints.NewManager(client, "service/"+s.Name)
	if err != nil {
		return err
	}
	var addr string
	if s.IsHost {
		addr = netx.GetOutboundIP() + ":" + strconv.Itoa(s.Port)
	} else {
		addr = s.Name + ":" + strconv.Itoa(s.Port)
	}
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
			// 续约过程打印debug日志就行
			s.L.Debug(kaResp.String())
		}
	}()
	return nil
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
	if s.client != nil {
		err := s.client.Close()
		if err != nil {
			return err
		}
	}
	s.Server.GracefulStop()
	return nil
}
