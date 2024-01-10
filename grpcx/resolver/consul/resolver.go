package consul

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	consulapi "github.com/hashicorp/consul/api"
	"google.golang.org/grpc/resolver"
)

// 需要实现 Resolver Builder 接口
type consulResolverBuilder struct {
	client *consulapi.Client
}

// Build creates a new Resolver for Consul.
func (b *consulResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	serviceName := target.URL.Path
	if serviceName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "resolver: missing service name in target")
	}
	serviceName = strings.TrimPrefix(serviceName, "/")
	r := &consulResolver{
		target:      target,
		client:      b.client,
		serviceName: serviceName,
		cc:          cc,
		ctx:         context.Background(),
		cancel:      func() {},
		instances:   make(map[string]struct{}),
	}
	r.ctx, r.cancel = context.WithCancel(r.ctx)
	go r.watch()
	return r, nil
}

// Scheme returns the scheme for Consul Resolver.
func (b *consulResolverBuilder) Scheme() string {
	return "consul"
}

// NewBuilder creates a new resolver builder for Consul.
func NewBuilder(client *consulapi.Client) (resolver.Builder, error) {
	return &consulResolverBuilder{
		client: client,
	}, nil
}

type consulResolver struct {
	client      *consulapi.Client
	serviceName string
	cc          resolver.ClientConn
	target      resolver.Target
	ctx         context.Context
	cancel      context.CancelFunc
	instances   map[string]struct{}
	mu          sync.Mutex
}

// watch continuously queries Consul for service instances and updates the resolver state.
func (r *consulResolver) watch() {
	queryOptions := &consulapi.QueryOptions{
		WaitIndex: 0,
	}

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		instances, meta, err := r.getInstances(queryOptions)
		if err != nil {
			fmt.Printf("Error retrieving instances from Consul: %v\n", err)
			time.Sleep(1 * time.Second) // Wait before retrying
			continue
		}

		r.updateInstances(instances)
		queryOptions.WaitIndex = meta.LastIndex
	}
}

// getInstances queries Consul for service instances.
func (r *consulResolver) getInstances(queryOptions *consulapi.QueryOptions) ([]*consulapi.ServiceEntry, *consulapi.QueryMeta, error) {
	entries, meta, err := r.client.Health().Service(r.serviceName, "", true, queryOptions)
	if err != nil {
		return nil, nil, err
	}
	return entries, meta, nil
}

// updateInstances processes obtained instances, updates resolver state, and removes instances that are no longer available.
func (r *consulResolver) updateInstances(instances []*consulapi.ServiceEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var updatedInstances []resolver.Address
	for _, entry := range instances {
		address := entry.Service.Address
		if address == "" {
			address = entry.Node.Address
		}
		port := entry.Service.Port

		instanceAddr := fmt.Sprintf("%s:%d", address, port)

		r.instances[instanceAddr] = struct{}{}

		updatedInstances = append(updatedInstances, resolver.Address{
			Addr:     instanceAddr,
			Metadata: entry.Service.Meta,
		})
	}

	r.cc.UpdateState(resolver.State{Addresses: updatedInstances})

	for addr := range r.instances {
		found := false
		for _, entry := range instances {
			address := entry.Service.Address
			if address == "" {
				address = entry.Node.Address
			}
			port := entry.Service.Port

			instanceAddr := fmt.Sprintf("%s:%d", address, port)
			if addr == instanceAddr {
				found = true
				break
			}
		}
		if !found {
			delete(r.instances, addr)
		}
	}
}

// ResolveNow is a no-op here.
// It's just a hint, resolver can ignore this if it's not necessary.
func (r *consulResolver) ResolveNow(resolver.ResolveNowOptions) {}

// Close cancels the context and waits for the watch goroutine to finish.
func (r *consulResolver) Close() {
	r.cancel()
}
