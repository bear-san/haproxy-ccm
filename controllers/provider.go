package controllers

import (
	haproxyv1 "github.com/bear-san/haproxy-configurator/pkg/haproxy/v1"
	"google.golang.org/grpc"
	cloudprovider "k8s.io/cloud-provider"
)

type Provider struct {
	cloudprovider.Interface
	HAProxyClient haproxyv1.HAProxyManagerServiceClient
	Connection    *grpc.ClientConn
}

func (p *Provider) Initialize(_ cloudprovider.ControllerClientBuilder, _ <-chan struct{}) {

}

func (p *Provider) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return &ServiceController{
		HAProxyClient: p.HAProxyClient,
	}, true
}

func (p *Provider) Instances() (cloudprovider.Instances, bool) {
	return nil, false
}

func (p *Provider) InstancesV2() (cloudprovider.InstancesV2, bool) {
	return nil, false
}

func (p *Provider) Zones() (cloudprovider.Zones, bool) {
	return nil, false
}

func (p *Provider) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

func (p *Provider) HasClusterID() bool {
	return true
}
