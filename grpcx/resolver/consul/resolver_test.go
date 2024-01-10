package consul

import (
	"context"
	"testing"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/wkRonin/toolkit/grpcx"
	"github.com/wkRonin/toolkit/grpcx/server_example"
	"github.com/wkRonin/toolkit/logger"
)

type ConsulTestSuite struct {
	suite.Suite
	consulAddr string
	l          logger.Logger
}

func (s *ConsulTestSuite) SetupSuite() {
	s.consulAddr = "10.0.0.8:18500"
	s.l = &logger.NopLogger{}
}

func (s *ConsulTestSuite) TestConsulResolverClient() {
	client, err := consulapi.NewClient(&consulapi.Config{
		Address: s.consulAddr,
	})
	require.NoError(s.T(), err)
	bd, err := NewBuilder(client)
	require.NoError(s.T(), err)
	cc, err := grpc.Dial("consul:///service/usertest",
		grpc.WithResolvers(bd),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	rpcClient := server_example.NewUserServiceClient(cc)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := rpcClient.GetById(ctx, &server_example.GetByIdRequest{
		Id: 123,
	})
	cancel()
	require.NoError(s.T(), err)
	s.T().Log(resp.User)

}

func (s *ConsulTestSuite) TestServer() {
	server := grpc.NewServer()
	server_example.RegisterUserServiceServer(server, &server_example.Server{
		// 用地址来标识
		Name: "10.0.0.8",
	})
	ss := &grpcx.Server{
		Server:          server,
		Port:            8090,
		UseConsulClient: true,
		ConsulAddrs:     s.consulAddr,
		Name:            "usertest",
		L:               s.l,
		IsHost:          true,
	}
	err := ss.Serve()
	require.NoError(s.T(), err)
}

func TestConsul(t *testing.T) {
	suite.Run(t, new(ConsulTestSuite))
}
