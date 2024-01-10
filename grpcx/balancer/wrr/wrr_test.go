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
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/wkRonin/toolkit/grpcx"
	"github.com/wkRonin/toolkit/grpcx/server_example"
	"github.com/wkRonin/toolkit/logger"
)

type EtcdTestSuite struct {
	suite.Suite
	etcdAddr []string
	l        logger.Logger
}

func (s *EtcdTestSuite) SetupSuite() {
	s.etcdAddr = []string{"10.0.0.8:12379"}
	s.l = &logger.NopLogger{}
}

func (s *EtcdTestSuite) TestCustomWRRClient() {
	client, err := etcdv3.New(etcdv3.Config{
		Endpoints: s.etcdAddr,
	})
	bd, err := resolver.NewBuilder(client)
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
	rpcClient := server_example.NewUserServiceClient(cc)
	for i := 0; i < 12; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		resp, err := rpcClient.GetById(ctx, &server_example.GetByIdRequest{
			Id: 123,
		})
		cancel()
		require.NoError(s.T(), err)
		s.T().Log(resp.User)
	}
}

func (s *EtcdTestSuite) TestServer() {
	go func() {
		s.startServer(8090, 20)
	}()

	go func() {
		s.startServer(8092, 30)
	}()
	s.startServer(8091, 10)
}

func (s *EtcdTestSuite) startServer(port int, weight int) {
	server := grpc.NewServer()
	server_example.RegisterUserServiceServer(server, &server_example.Server{
		// 用地址来标识
		Name: "10.0.0.8:" + strconv.Itoa(port),
	})
	ss := &grpcx.Server{
		Server:    server,
		Port:      port,
		EtcdAddrs: s.etcdAddr,
		Name:      "usertest",
		L:         s.l,
		IsHost:    true,
		Weight:    weight,
		UseWrr:    true,
	}
	err := ss.Serve()
	require.NoError(s.T(), err)
}

func TestEtcd(t *testing.T) {
	suite.Run(t, new(EtcdTestSuite))
}
