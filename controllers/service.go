package controllers

import (
	"context"
	"fmt"
	haproxyv3 "github.com/bear-san/haproxy-go/dataplane/v3"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
	"strings"
)

type ServiceController struct {
	cloudprovider.LoadBalancer
	HAProxyClient *haproxyv3.Client
}

func (s *ServiceController) UpdateLoadBalancer(_ context.Context, _ string, service *v1.Service, nodes []*v1.Node) error {
	klog.Info("Updating HAProxy LoadBalancer...")
	if _, err := s.reconcileLoadBalancer(context.Background(), service, nodes); err != nil {
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

	resourcePrefix := fmt.Sprintf("haproxy-%s-", service.UID)

	// delete all frontends and binds
	frontends, err := s.HAProxyClient.ListFrontend(*transaction.Id)
	if err != nil {
		klog.Errorf("list frontend error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}
		return err
	}

	for _, frontend := range frontends {
		if !strings.HasPrefix(*frontend.Name, resourcePrefix) {
			continue
		}

		binds, err := s.HAProxyClient.ListBind(*frontend.Name, *transaction.Id)
		if err != nil {
			klog.Errorf("list bind error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return err
		}

		for _, bind := range binds {
			err := s.HAProxyClient.DeleteBind(*bind.Name, *frontend.Name, *transaction.Id)
			if err != nil {
				klog.Errorf("delete bind error: %v", err.Error())
				if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
					klog.Errorf("close transaction error: %v", err.Error())
				}
				return err
			}
		}

		if err := s.HAProxyClient.DeleteFrontend(*frontend.Name, *transaction.Id); err != nil {
			klog.Errorf("delete frontend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return err
		}
	}

	// delete all backends and servers
	backends, err := s.HAProxyClient.ListBackend(*transaction.Id)
	if err != nil {
		klog.Errorf("list backend error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}
		return err
	}

	for _, backend := range backends {
		if !strings.HasPrefix(*backend.Name, resourcePrefix) {
			continue
		}

		servers, err := s.HAProxyClient.ListServer(*backend.Name, *transaction.Id)
		if err != nil {
			klog.Errorf("list server error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return err
		}

		for _, server := range servers {
			if err := s.HAProxyClient.DeleteServer(*server.Name, *backend.Name, *transaction.Id); err != nil {
				klog.Errorf("delete server error: %v", err.Error())
				if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
					klog.Errorf("close transaction error: %v", err.Error())
				}
				return err
			}
		}

		if err := s.HAProxyClient.DeleteBackend(*backend.Name, *transaction.Id); err != nil {
			klog.Errorf("delete backend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return err
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

func (s *ServiceController) EnsureLoadBalancer(_ context.Context, _ string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	klog.Info("Creating HAProxy LoadBalancer...")
	return s.reconcileLoadBalancer(context.Background(), service, nodes)
}

func (s *ServiceController) reconcileLoadBalancer(_ context.Context, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
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

	resourcePrefix := fmt.Sprintf("haproxy-%s", service.UID)

	backends, err := s.HAProxyClient.ListBackend(*transaction.Id)
	if err != nil {
		klog.Errorf("list backend error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}
		return nil, err
	}

	// delete all backends and servers
	for _, backend := range backends {
		if !strings.HasPrefix(*backend.Name, resourcePrefix) {
			continue
		}

		servers, err := s.HAProxyClient.ListServer(*backend.Name, *transaction.Id)
		if err != nil {
			klog.Errorf("list server error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return nil, err
		}

		for _, server := range servers {
			if err := s.HAProxyClient.DeleteServer(*server.Name, *backend.Name, *transaction.Id); err != nil {
				klog.Errorf("delete server error: %v", err.Error())
				if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
					klog.Errorf("close transaction error: %v", err.Error())
				}
				return nil, err
			}
		}
	}

	frontends, err := s.HAProxyClient.ListFrontend(*transaction.Id)
	if err != nil {
		klog.Errorf("list frontend error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}
		return nil, err
	}
	for _, frontend := range frontends {
		if !strings.HasPrefix(*frontend.Name, resourcePrefix) {
			continue
		}

		binds, err := s.HAProxyClient.ListBind(*frontend.Name, *transaction.Id)
		if err != nil {
			klog.Errorf("list bind error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return nil, err
		}

		for _, bind := range binds {
			err := s.HAProxyClient.DeleteBind(*bind.Name, *frontend.Name, *transaction.Id)
			if err != nil {
				klog.Errorf("delete bind error: %v", err.Error())

				if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
					klog.Errorf("close transaction error: %v", err.Error())
				}
				return nil, err
			}
		}
		if err := s.HAProxyClient.DeleteFrontend(*frontend.Name, *transaction.Id); err != nil {
			klog.Errorf("delete frontend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return nil, err
		}
	}

	// create a new backend and backend servers
	for _, port := range service.Spec.Ports {
		resourceName := fmt.Sprintf("%s-%s-%s", resourcePrefix, port.Name, port.Protocol)
		if _, err := s.HAProxyClient.AddBackend(haproxyv3.Backend{
			Balance: &haproxyv3.BackendBalance{
				Algorithm: haproxyv3.BACKEND_BALANCE_ALGORITHM_ROUNDROBIN,
			},
			Name: &resourceName,
			Mode: haproxyv3.BACKEND_MODE_TCP,
		}, *transaction.Id); err != nil {
			klog.Errorf("create backend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return nil, err
		}

		for i, node := range nodes {
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
			if _, err = s.HAProxyClient.AddServer(resourceName, *transaction.Id, newServer); err != nil {
				klog.Errorf("create server error: %v", err.Error())
				if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
					klog.Errorf("close transaction error: %v", err.Error())
				}
				return nil, err
			}
		}
	}

	// Create new frontend if not exists
	for _, port := range service.Spec.Ports {
		resourceName := fmt.Sprintf("%s-%s-%s", resourcePrefix, port.Name, port.Protocol)
		mode := haproxyv3.FRONTEND_MODE_TCP
		if _, err := s.HAProxyClient.AddFrontend(haproxyv3.Frontend{
			DefaultBackend: &resourceName,
			Name:           &resourceName,
			Mode:           &mode,
		}, *transaction.Id); err != nil {
			klog.Errorf("create frontend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return nil, err
		}

		for _, ip := range service.Spec.ExternalIPs {
			bindName := fmt.Sprintf("%s-%s-%s", resourcePrefix, ip, port.Protocol)
			portNum := int(port.Port)
			if _, err := s.HAProxyClient.AddBind(resourceName, *transaction.Id, haproxyv3.Bind{
				Name:    &bindName,
				Address: &ip,
				Port:    &portNum,
				V4V6:    nil,
				V6Only:  nil,
			}); err != nil {
				klog.Errorf("create bind error: %v", err.Error())
				if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(*transaction.Id); closeTransactionErr != nil {
					klog.Errorf("close transaction error: %v", err.Error())
				}
				return nil, err
			}
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
