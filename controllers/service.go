package controllers

import (
	"context"
	"fmt"
	"github.com/bear-san/haproxy-ccm/pkg/haproxy"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
)

type ServiceController struct {
	cloudprovider.LoadBalancer
}

func (s *ServiceController) GetLoadBalancerName(_ context.Context, _ string, service *v1.Service) string {
	return fmt.Sprintf("haproxy-%s", service.UID)
}

func (s *ServiceController) GetLoadBalancer(_ context.Context, _ string, service *v1.Service) (status *v1.LoadBalancerStatus, exists bool, err error) {
	frontends, err := haproxy.ListFrontend()
	if err != nil {
		return nil, false, err
	}

	for _, frontend := range frontends {
		if frontend.Name == fmt.Sprintf("frontend-%s", service.UID) {
			binds, err := haproxy.ListBind(fmt.Sprintf("frontend-%s", service.UID))
			if err != nil {
				return nil, false, err
			}

			return &v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{
					{
						IP: binds[0].Address,
					},
				},
			}, true, nil
		}
	}

	return nil, false, nil
}

func (s *ServiceController) EnsureLoadBalancerDeleted(_ context.Context, _ string, service *v1.Service) error {
	binds, err := haproxy.ListBind(fmt.Sprintf("frontend-%s", service.UID))
	if err != nil {
		return err
	}

	for _, bind := range binds {
		err := haproxy.DeleteBind(bind.Name, fmt.Sprintf("frontend-%s", service.UID), nil)
		if err != nil {
			return err
		}
	}

	haproxy.DeleteFrontend(fmt.Sprintf("frontend-%s", service.UID), nil)

	servers, err := haproxy.ListServer(fmt.Sprintf("backend-%s", service.UID))
	if err != nil {
		return err
	}
	for _, server := range servers {
		err := haproxy.DeleteServer(server.Name, fmt.Sprintf("backend-%s", service.UID), nil)
		if err != nil {
			return err
		}
	}

	haproxy.DeleteBackend(fmt.Sprintf("backend-%s", service.UID), nil)

	return nil
}

func (s *ServiceController) EnsureLoadBalancer(_ context.Context, _ string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	newBackend := haproxy.Backend{
		Balance: struct {
			Algorithm string `json:"algorithm"`
		}(struct{ Algorithm string }{
			Algorithm: "roundrobin",
		}),
		Mode: "tcp",
		Name: fmt.Sprintf("backend-%s", service.UID),
	}
	haproxy.CreateBackend(newBackend, nil)

	for _, node := range nodes {
		for _, port := range service.Spec.Ports {
			newServer := haproxy.Server{
				Address: node.Name,
				Port:    int(port.NodePort),
				Name:    fmt.Sprintf("server-%s-%s-%d", service.UID, node.Name, port.NodePort),
			}
			haproxy.CreateServer(newBackend.Name, newServer, nil)
		}
	}

	newFrontend := haproxy.Frontend{
		DefaultBackend: newBackend.Name,
		Mode:           "tcp",
		Name:           fmt.Sprintf("frontend-%s", service.UID),
		Tcplog:         false,
	}
	haproxy.CreateFrontend(newFrontend, nil)

	ipAddr := service.Spec.LoadBalancerIP
	if ipAddr == "" {
		// Not implemented
		return nil, fmt.Errorf("auto assign IP not implemented")
	}

	for _, port := range service.Spec.Ports {
		newBind := haproxy.Bind{
			Address: ipAddr,
			Port:    int(port.Port),
			Name:    fmt.Sprintf("bind-%s-%d", service.UID, port.Port),
		}

		haproxy.CreateBind(newFrontend.Name, newBind, nil)
	}

	return &v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{
			{
				IP: ipAddr,
			},
		},
	}, nil
}
