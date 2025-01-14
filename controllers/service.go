package controllers

import (
	"context"
	"fmt"
	"github.com/bear-san/haproxy-ccm/pkg/haproxy"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

type ServiceController struct {
	cloudprovider.LoadBalancer
}

func (s *ServiceController) UpdateLoadBalancer(_ context.Context, _ string, service *v1.Service, nodes []*v1.Node) error {
	klog.Info("Updating HAProxy LoadBalancer...")
	if len(service.Spec.ExternalIPs) == 0 {
		// Not implemented
		return fmt.Errorf("auto assign IP not implemented")
	}

	transaction, err := haproxy.CreateTransaction()
	if err != nil {
		return err
	}

	// Delete all servers and binds
	servers, err := haproxy.ListServer(fmt.Sprintf("backend-%s", service.UID))
	if err != nil {
		return err
	}

	for _, server := range servers {
		err := haproxy.DeleteServer(server.Name, fmt.Sprintf("backend-%s", service.UID), transaction)
		if err != nil {
			return err
		}
	}

	binds, err := haproxy.ListBind(fmt.Sprintf("frontend-%s", service.UID))
	if err != nil {
		return err
	}
	for _, bind := range binds {
		err := haproxy.DeleteBind(bind.Name, fmt.Sprintf("frontend-%s", service.UID), transaction)
		if err != nil {
			return err
		}
	}

	// Create new servers and binds
	for i, p := range service.Spec.ExternalIPs {
		for _, port := range service.Spec.Ports {
			// create backend.
			// backend is a collection of servers
			newBackend := haproxy.Backend{
				Balance: struct {
					Algorithm string `json:"algorithm"`
				}(struct{ Algorithm string }{
					Algorithm: "roundrobin",
				}),
				Mode: "tcp",
				Name: fmt.Sprintf("backend-%s-%s-%d", service.UID, port.Name, i),
			}

			err = haproxy.CreateBackend(newBackend, transaction)
			if err != nil {
				return err
			}

			// create server.
			// server is a pair of node and port links to the backend.
			for _, node := range nodes {
				nodeIp := ""
				for _, address := range node.Status.Addresses {
					if address.Type == v1.NodeInternalIP {
						nodeIp = address.Address
						break
					}
				}

				// skip if node doesn't have internal IP
				if nodeIp == "" {
					continue
				}

				newServer := haproxy.Server{
					Address: nodeIp,
					Port:    int(port.NodePort),
					Name:    fmt.Sprintf("server-%s-%s-%d-%d", service.UID, node.Name, port.NodePort, i),
				}
				err = haproxy.CreateServer(newBackend.Name, newServer, transaction)
				if err != nil {
					return err
				}
			}

			// create frontend.
			// frontend is a collection of binds
			newFrontend := haproxy.Frontend{
				DefaultBackend: newBackend.Name,
				Mode:           "tcp",
				Name:           fmt.Sprintf("frontend-%s-%s-%d", service.UID, port.Name, i),
			}
			err = haproxy.CreateFrontend(newFrontend, transaction)
			if err != nil {
				return err
			}

			// create bind.
			// bind is a pair of ip and port to expose the service to the external networks.
			newBind := haproxy.Bind{
				Address: p,
				Port:    int(port.Port),
				Name:    fmt.Sprintf("bind-%s-%d-%d", service.UID, port.Port, i),
			}

			err = haproxy.CreateBind(newFrontend.Name, newBind, transaction)
			if err != nil {
				return err
			}
		}
	}

	return haproxy.CommitTransaction(transaction.Id)
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
		for i, p := range service.Spec.ExternalIPs {
			for _, port := range service.Spec.Ports {
				if frontend.Name == fmt.Sprintf("frontend-%s-%s-%d", service.UID, port.Name, i) {
					return &v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{
								IP: p,
							},
						},
					}, true, nil
				}
			}
		}
	}

	return nil, false, nil
}

func (s *ServiceController) EnsureLoadBalancerDeleted(_ context.Context, _ string, service *v1.Service) error {
	klog.Info("Deleting HAProxy LoadBalancer...")
	transaction, err := haproxy.CreateTransaction()
	if err != nil {
		return err
	}

	// Delete all servers and binds
	for i := range service.Spec.ExternalIPs {
		for _, port := range service.Spec.Ports {
			err = haproxy.DeleteFrontend(fmt.Sprintf("frontend-%s-%s-%d", service.UID, port.Name, i), transaction)
			if err != nil {
				return err
			}

			err = haproxy.DeleteBackend(fmt.Sprintf("backend-%s-%s-%d", service.UID, port.Name, i), transaction)
			if err != nil {
				return err
			}
		}
	}

	return haproxy.CommitTransaction(transaction.Id)
}

func (s *ServiceController) EnsureLoadBalancer(_ context.Context, _ string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	klog.Info("Creating HAProxy LoadBalancer...")
	if len(service.Spec.ExternalIPs) == 0 {
		// Not implemented
		return nil, fmt.Errorf("auto assign IP not implemented")
	}

	newStatus := v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{},
	}

	transaction, err := haproxy.CreateTransaction()
	if err != nil {
		return nil, err
	}

	for i, p := range service.Spec.ExternalIPs {
		for _, port := range service.Spec.Ports {
			// create backend.
			// backend is a collection of servers
			newBackend := haproxy.Backend{
				Balance: struct {
					Algorithm string `json:"algorithm"`
				}(struct{ Algorithm string }{
					Algorithm: "roundrobin",
				}),
				Mode: "tcp",
				Name: fmt.Sprintf("backend-%s-%s-%d", service.UID, port.Name, i),
			}

			// delete current backend if exists
			currentBackend, err := haproxy.GetBackend(newBackend.Name)
			if err == nil {
				err = haproxy.DeleteBackend(currentBackend.Name, transaction)
			}

			err = haproxy.CreateBackend(newBackend, transaction)
			if err != nil {
				return nil, err
			}

			// create server.
			// server is a pair of node and port links to the backend.
			for _, node := range nodes {
				nodeIp := ""
				for _, address := range node.Status.Addresses {
					if address.Type == v1.NodeInternalIP {
						nodeIp = address.Address
						break
					}
				}

				// skip if node doesn't have internal IP
				if nodeIp == "" {
					continue
				}

				newServer := haproxy.Server{
					Address: nodeIp,
					Port:    int(port.NodePort),
					Name:    fmt.Sprintf("server-%s-%s-%d-%d", service.UID, node.Name, port.NodePort, i),
				}

				// delete current server if exists
				currentServer, err := haproxy.GetServer(newServer.Name, newBackend.Name)
				if err == nil {
					err = haproxy.DeleteServer(currentServer.Name, newBackend.Name, transaction)
				}
				err = haproxy.CreateServer(newBackend.Name, newServer, transaction)
				if err != nil {
					return nil, err
				}
			}

			// create frontend.
			// frontend is a collection of binds
			newFrontend := haproxy.Frontend{
				DefaultBackend: newBackend.Name,
				Mode:           "tcp",
				Name:           fmt.Sprintf("frontend-%s-%s-%d", service.UID, port.Name, i),
			}

			// delete current frontend if exists
			currentFrontend, err := haproxy.GetFrontend(newFrontend.Name)
			if err == nil {
				err = haproxy.DeleteFrontend(currentFrontend.Name, transaction)
			}
			err = haproxy.CreateFrontend(newFrontend, transaction)
			if err != nil {
				return nil, err
			}

			// create bind.
			// bind is a pair of ip and port to expose the service to the external networks.
			newBind := haproxy.Bind{
				Address: p,
				Port:    int(port.Port),
				Name:    fmt.Sprintf("bind-%s-%d-%d", service.UID, port.Port, i),
			}

			// delete current bind if exists
			currentBind, err := haproxy.GetBind(newBind.Name, newFrontend.Name)
			if err == nil {
				err = haproxy.DeleteBind(currentBind.Name, newFrontend.Name, transaction)
			}

			err = haproxy.CreateBind(newFrontend.Name, newBind, transaction)
			if err != nil {
				return nil, err
			}
		}

		newStatus.Ingress = append(newStatus.Ingress, v1.LoadBalancerIngress{
			IP: p,
		})
	}

	err = haproxy.CommitTransaction(transaction.Id)
	if err != nil {
		return nil, err
	}

	return &newStatus, nil
}
