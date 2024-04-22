package controllers

import (
	cloudprovider "k8s.io/cloud-provider"
)

type Provider struct {
	cloudprovider.Interface
}

func (p *Provider) Initialize(_ cloudprovider.ControllerClientBuilder, _ <-chan struct{}) {

}

func (p *Provider) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return &ServiceController{}, true
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
