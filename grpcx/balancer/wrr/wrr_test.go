package wrr

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/wkRonin/toolkit/grpcx/balancer/wrr/example"
	"github.com/wkRonin/toolkit/netx"
)

type EtcdTestSuite struct {
	suite.Suite
	client *etcdv3.Client
}

func (s *EtcdTestSuite) SetupSuite() {
	client, err := etcdv3.New(etcdv3.Config{
		Endpoints: []string{"10.0.0.8:12379"},
	})
	require.NoError(s.T(), err)
	s.client = client
}

func (s *EtcdTestSuite) TestCustomRoundRobinClient() {
	bd, err := resolver.NewBuilder(s.client)
	require.NoError(s.T(), err)
	svcCfg := `
{
    "loadBalancingConfig": [
        {
            "custom_wrr": {}
        }
    ]
}
`
	cc, err := grpc.Dial("etcd:///service/usertest",
		grpc.WithResolvers(bd),
		// 在这里使用的负载均衡器
		grpc.WithDefaultServiceConfig(svcCfg),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := example.NewUserServiceClient(cc)
	for i := 0; i < 12; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		resp, err := client.GetById(ctx, &example.GetByIdRequest{
			Id: 123,
		})
		cancel()
		require.NoError(s.T(), err)
		s.T().Log(resp.User)
	}
}

func (s *EtcdTestSuite) TestServer() {
	go func() {
		s.startServer(":8090", 20)
	}()

	go func() {
		s.startServer(":8092", 30)
	}()
	s.startServer(":8091", 10)
}

func (s *EtcdTestSuite) startServer(addr string, weight int) {
	l, err := net.Listen("tcp", addr)
	require.NoError(s.T(), err)

	em, err := endpoints.NewManager(s.client, "service/usertest")
	require.NoError(s.T(), err)
	addr = netx.GetOutboundIP() + addr
	key := "service/usertest/" + addr
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	var ttl int64 = 30
	leaseResp, err := s.client.Grant(ctx, ttl)
	require.NoError(s.T(), err)
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = em.AddEndpoint(ctx, key, endpoints.Endpoint{
		Addr: addr,
		Metadata: map[string]any{
			"weight": weight,
		},
	}, etcdv3.WithLease(leaseResp.ID))
	require.NoError(s.T(), err)

	kaCtx, kaCancel := context.WithCancel(context.Background())
	go func() {
		// 操作续约
		_, err1 := s.client.KeepAlive(kaCtx, leaseResp.ID)
		require.NoError(s.T(), err1)
	}()

	server := grpc.NewServer()
	example.RegisterUserServiceServer(server, &example.Server{
		// 用地址来标识
		Name: addr,
	})
	err = server.Serve(l)
	s.T().Log(err)
	// 你要退出了，正常退出
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 我要先取消续约
	kaCancel()
	// 退出阶段，先从注册中心里面删了自己
	err = em.DeleteEndpoint(ctx, key)
	// 关掉客户端
	s.client.Close()
	server.GracefulStop()
}

func TestEtcd(t *testing.T) {
	suite.Run(t, new(EtcdTestSuite))
}
