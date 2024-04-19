package controllers

import (
	"context"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
)

type ServiceController struct {
	cloudprovider.LoadBalancer
}

func (c *ServiceController) GetLoadBalancer(ctx context.Context, service *v1.Service) (*v1.LoadBalancerStatus, bool, error) {

	return nil, false, nil
}
