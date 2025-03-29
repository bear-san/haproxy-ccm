package controllers

import (
	"context"
	"fmt"
	haproxyv3 "github.com/bear-san/haproxy-go/dataplane/v3"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

type ServiceController struct {
	cloudprovider.LoadBalancer
	HAProxyClient *haproxyv3.Client
}

func (s *ServiceController) UpdateLoadBalancer(_ context.Context, _ string, service *v1.Service, nodes []*v1.Node) error {
	klog.Info("Updating HAProxy LoadBalancer...")
	if len(service.Spec.ExternalIPs) == 0 {
		// Not implemented
		return fmt.Errorf("auto assign IP not implemented")
	}

	currentVersion, err := s.HAProxyClient.GetVersion()
	if err != nil {
		klog.Errorf("get current version error: %v", err.Error())
		return err
	}
	transaction, err := s.HAProxyClient.CreateTransaction(*currentVersion)
	if err != nil {
		klog.Errorf("create transaction error: %v", err.Error())
		return err
	}

	backendName := fmt.Sprintf("backend-%s", service.UID)

	// Delete all servers and binds
	servers, err := s.HAProxyClient.ListServer(backendName, *transaction.Id)
	if err != nil {
		klog.Errorf("list server error: %v", err.Error())

		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}

		return err
	}

	for _, server := range servers {
		err := s.HAProxyClient.DeleteServer(*server.Name, backendName, *transaction.Id)
		if err != nil {
			klog.Errorf("delete server error: %v", err.Error())

			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}

			return err
		}
	}

	frontendName := fmt.Sprintf("frontend-%s", service.UID)
	binds, err := s.HAProxyClient.ListBind(frontendName, *transaction.Id)
	if err != nil {
		klog.Errorf("list bind error: %v", err.Error())
		return err
	}
	for _, bind := range binds {
		err := s.HAProxyClient.DeleteBind(*bind.Name, frontendName, *transaction.Id)
		if err != nil {
			klog.Errorf("delete bind error: %v", err.Error())

			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return err
		}
	}

	// Create new backend if not exists
	if _, err := s.HAProxyClient.GetBackend(backendName, *transaction.Id); err != nil {
		// backend is a collection of servers
		newBackend := haproxyv3.Backend{
			Balance: &haproxyv3.BackendBalance{
				Algorithm: haproxyv3.BACKEND_BALANCE_ALGORITHM_ROUNDROBIN,
			},
			Mode: "tcp",
			Name: &backendName,
		}

		_, err = s.HAProxyClient.AddBackend(newBackend, *transaction.Id)
		if err != nil {
			klog.Errorf("create backend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return err
		}
	}

	// Create new frontend if not exists
	if _, err := s.HAProxyClient.GetFrontend(frontendName, *transaction.Id); err != nil {
		// frontend is a collection of binds
		mode := haproxyv3.FRONTEND_MODE_TCP
		newFrontend := haproxyv3.Frontend{
			DefaultBackend: &backendName,
			Mode:           &mode,
			Name:           &frontendName,
		}
		if _, err = s.HAProxyClient.AddFrontend(newFrontend, *transaction.Id); err != nil {
			klog.Errorf("create frontend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return err
		}
	}

	// Create new servers and binds
	for i, p := range service.Spec.ExternalIPs {
		for _, port := range service.Spec.Ports {

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
				nodePort := int(port.NodePort)
				serverName := fmt.Sprintf("server-%s-%s-%d-%d", service.UID, node.Name, port.NodePort, i)
				newServer := haproxyv3.Server{
					Address: &nodeIp,
					Port:    &nodePort,
					Name:    &serverName,
				}
				_, err = s.HAProxyClient.AddServer(backendName, *transaction.Id, newServer)
				if err != nil {
					return err
				}
			}

			// create bind.
			// bind is a pair of ip and port to expose the service to the external networks.
			bindName := fmt.Sprintf("bind-%s-%d-%d", service.UID, port.Port, i)
			portNumber := int(port.Port)
			newBind := haproxyv3.Bind{
				Address: &p,
				Port:    &portNumber,
				Name:    &bindName,
			}

			if _, err = s.HAProxyClient.AddBind(frontendName, *transaction.Id, newBind); err != nil {
				klog.Errorf("create bind error: %v", err.Error())
				if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
					klog.Errorf("close transaction error: %v", err.Error())
				}
				return err
			}
		}
	}

	if _, err := s.HAProxyClient.CommitTransaction(*transaction.Id); err != nil {
		klog.Errorf("commit transaction error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}
		return err
	}

	return nil
}

func (s *ServiceController) GetLoadBalancerName(_ context.Context, _ string, service *v1.Service) string {
	return fmt.Sprintf("haproxy-%s", service.UID)
}

func (s *ServiceController) GetLoadBalancer(_ context.Context, _ string, service *v1.Service) (status *v1.LoadBalancerStatus, exists bool, err error) {
	return &service.Status.LoadBalancer, true, nil
}

func (s *ServiceController) EnsureLoadBalancerDeleted(_ context.Context, _ string, service *v1.Service) error {
	klog.Info("Deleting HAProxy LoadBalancer...")
	currentVersion, err := s.HAProxyClient.GetVersion()
	if err != nil {
		klog.Errorf("get current version error: %v", err.Error())
		return err
	}
	transaction, err := s.HAProxyClient.CreateTransaction(*currentVersion)
	if err != nil {
		klog.Errorf("create transaction error: %v", err.Error())
		return err
	}

	backendName := fmt.Sprintf("backend-%s", service.UID)
	frontendName := fmt.Sprintf("frontend-%s", service.UID)

	// Delete all servers and binds
	servers, err := s.HAProxyClient.ListServer(backendName, *transaction.Id)
	if err != nil {
		klog.Errorf("list servers error: %v", err.Error())
	}
	for _, server := range servers {
		if err := s.HAProxyClient.DeleteServer(*server.Name, backendName, *transaction.Id); err != nil {
			klog.Errorf("delete server error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}

			return err
		}
	}
	binds, err := s.HAProxyClient.ListBind(frontendName, *transaction.Id)
	if err != nil {
		klog.Errorf("list servers error: %v", err.Error())
	}
	for _, bind := range binds {
		if err := s.HAProxyClient.DeleteBind(*bind.Name, frontendName, *transaction.Id); err != nil {
			klog.Errorf("delete bind error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}

			return err
		}
	}

	if err := s.HAProxyClient.DeleteBackend(backendName, *transaction.Id); err != nil {
		klog.Errorf("delete backend error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}

		return err
	}

	if err := s.HAProxyClient.DeleteFrontend(frontendName, *transaction.Id); err != nil {
		klog.Errorf("delete frontend error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}

		return err
	}

	if _, err := s.HAProxyClient.CommitTransaction(*transaction.Id); err != nil {
		klog.Errorf("commit transaction error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}

		return err
	}

	return nil
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

	if len(service.Spec.ExternalIPs) == 0 {
		// Not implemented
		return nil, fmt.Errorf("auto assign IP not implemented")
	}

	currentVersion, err := s.HAProxyClient.GetVersion()
	if err != nil {
		klog.Errorf("get current version error: %v", err.Error())
		return nil, err
	}
	transaction, err := s.HAProxyClient.CreateTransaction(*currentVersion)
	if err != nil {
		klog.Errorf("create transaction error: %v", err.Error())
		return nil, err
	}

	backendName := fmt.Sprintf("backend-%s", service.UID)

	// Delete all servers and binds
	servers, err := s.HAProxyClient.ListServer(backendName, *transaction.Id)
	if err != nil {
		klog.Errorf("list server error: %v", err.Error())

		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}

		return nil, err
	}

	for _, server := range servers {
		err := s.HAProxyClient.DeleteServer(*server.Name, backendName, *transaction.Id)
		if err != nil {
			klog.Errorf("delete server error: %v", err.Error())

			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}

			return nil, err
		}
	}

	frontendName := fmt.Sprintf("frontend-%s", service.UID)
	binds, err := s.HAProxyClient.ListBind(frontendName, *transaction.Id)
	if err != nil {
		klog.Errorf("list bind error: %v", err.Error())
		return nil, err
	}
	for _, bind := range binds {
		err := s.HAProxyClient.DeleteBind(*bind.Name, frontendName, *transaction.Id)
		if err != nil {
			klog.Errorf("delete bind error: %v", err.Error())

			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return nil, err
		}
	}

	// Create new backend if not exists
	if _, err := s.HAProxyClient.GetBackend(backendName, *transaction.Id); err != nil {
		// backend is a collection of servers
		newBackend := haproxyv3.Backend{
			Balance: &haproxyv3.BackendBalance{
				Algorithm: haproxyv3.BACKEND_BALANCE_ALGORITHM_ROUNDROBIN,
			},
			Mode: "tcp",
			Name: &backendName,
		}

		_, err = s.HAProxyClient.AddBackend(newBackend, *transaction.Id)
		if err != nil {
			klog.Errorf("create backend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return nil, err
		}
	}

	// Create new frontend if not exists
	if _, err := s.HAProxyClient.GetFrontend(frontendName, *transaction.Id); err != nil {
		// frontend is a collection of binds
		mode := haproxyv3.FRONTEND_MODE_TCP
		newFrontend := haproxyv3.Frontend{
			DefaultBackend: &backendName,
			Mode:           &mode,
			Name:           &frontendName,
		}
		if _, err = s.HAProxyClient.AddFrontend(newFrontend, *transaction.Id); err != nil {
			klog.Errorf("create frontend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return nil, err
		}
	}

	// Create new servers and binds
	for i, p := range service.Spec.ExternalIPs {
		for _, port := range service.Spec.Ports {

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
				nodePort := int(port.NodePort)
				serverName := fmt.Sprintf("server-%s-%s-%d-%d", service.UID, node.Name, port.NodePort, i)
				newServer := haproxyv3.Server{
					Address: &nodeIp,
					Port:    &nodePort,
					Name:    &serverName,
				}
				_, err = s.HAProxyClient.AddServer(backendName, *transaction.Id, newServer)
				if err != nil {
					return nil, err
				}
			}

			// create bind.
			// bind is a pair of ip and port to expose the service to the external networks.
			bindName := fmt.Sprintf("bind-%s-%d-%d", service.UID, port.Port, i)
			portNumber := int(port.Port)
			newBind := haproxyv3.Bind{
				Address: &p,
				Port:    &portNumber,
				Name:    &bindName,
			}

			if _, err = s.HAProxyClient.AddBind(frontendName, *transaction.Id, newBind); err != nil {
				klog.Errorf("create bind error: %v", err.Error())
				if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
					klog.Errorf("close transaction error: %v", err.Error())
				}
				return nil, err
			}

			newStatus.Ingress = append(newStatus.Ingress, v1.LoadBalancerIngress{
				IP: p,
			})
		}
	}

	if _, err := s.HAProxyClient.CommitTransaction(*transaction.Id); err != nil {
		klog.Errorf("commit transaction error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}
		return nil, err
	}

	return &newStatus, nil
}
