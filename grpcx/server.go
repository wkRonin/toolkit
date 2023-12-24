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
	kaCancel  func()
	em        endpoints.Manager
	client    *etcdv3.Client
	key       string
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
	addr := netx.GetOutboundIP() + ":" + strconv.Itoa(s.Port)
	key := "service/" + s.Name + "/" + addr
	s.key = key
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// 续租时间暂时不需要做成可配置，默认30秒即可
	var ttl int64 = 30
	leaseResp, err := client.Grant(ctx, ttl)

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = em.AddEndpoint(ctx, key, endpoints.Endpoint{
		Addr: addr,
	}, etcdv3.WithLease(leaseResp.ID))

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
