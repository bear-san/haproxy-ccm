package controllers

import (
	"context"
	"fmt"
	haproxyv1 "github.com/bear-san/haproxy-configurator/pkg/haproxy/v1"
	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
	"strings"
)

type ServiceController struct {
	cloudprovider.LoadBalancer
	HAProxyClient haproxyv1.HAProxyManagerServiceClient
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

func (s *ServiceController) EnsureLoadBalancerDeleted(ctx context.Context, _ string, service *v1.Service) error {
	klog.Info("Deleting HAProxy LoadBalancer...")
	versionResp, err := s.HAProxyClient.GetVersion(ctx, &haproxyv1.GetVersionRequest{})
	if err != nil {
		klog.Errorf("get current version error: %v", err.Error())
		return err
	}
	transactionResp, err := s.HAProxyClient.CreateTransaction(ctx, &haproxyv1.CreateTransactionRequest{
		Version: versionResp.Version,
	})
	if err != nil {
		klog.Errorf("create transaction error: %v", err.Error())
		return err
	}

	resourcePrefix := fmt.Sprintf("haproxy-%s-", service.UID)

	// delete all frontends and binds
	frontendsResp, err := s.HAProxyClient.ListFrontends(ctx, &haproxyv1.ListFrontendsRequest{
		TransactionId: transactionResp.Transaction.Id,
	})
	if err != nil {
		klog.Errorf("list frontend error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
			TransactionId: transactionResp.Transaction.Id,
		}); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}
		return err
	}

	for _, frontend := range frontendsResp.Frontends {
		if !strings.HasPrefix(frontend.Name, resourcePrefix) {
			continue
		}

		bindsResp, err := s.HAProxyClient.ListBinds(ctx, &haproxyv1.ListBindsRequest{
			FrontendName:  frontend.Name,
			TransactionId: transactionResp.Transaction.Id,
		})
		if err != nil {
			klog.Errorf("list bind error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
				TransactionId: transactionResp.Transaction.Id,
			}); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return err
		}

		for _, bind := range bindsResp.Binds {
			_, err := s.HAProxyClient.DeleteBind(ctx, &haproxyv1.DeleteBindRequest{
				Name:          bind.Name,
				FrontendName:  frontend.Name,
				TransactionId: transactionResp.Transaction.Id,
			})
			if err != nil {
				klog.Errorf("delete bind error: %v", err.Error())
				if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
					TransactionId: transactionResp.Transaction.Id,
				}); closeTransactionErr != nil {
					klog.Errorf("close transaction error: %v", err.Error())
				}
				return err
			}
		}

		_, err = s.HAProxyClient.DeleteFrontend(ctx, &haproxyv1.DeleteFrontendRequest{
			Name:          frontend.Name,
			TransactionId: transactionResp.Transaction.Id,
		})
		if err != nil {
			klog.Errorf("delete frontend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
				TransactionId: transactionResp.Transaction.Id,
			}); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return err
		}
	}

	// delete all backends and servers
	backendsResp, err := s.HAProxyClient.ListBackends(ctx, &haproxyv1.ListBackendsRequest{
		TransactionId: transactionResp.Transaction.Id,
	})
	if err != nil {
		klog.Errorf("list backend error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
			TransactionId: transactionResp.Transaction.Id,
		}); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}
		return err
	}

	for _, backend := range backendsResp.Backends {
		if !strings.HasPrefix(backend.Name, resourcePrefix) {
			continue
		}

		serversResp, err := s.HAProxyClient.ListServers(ctx, &haproxyv1.ListServersRequest{
			BackendName:   backend.Name,
			TransactionId: transactionResp.Transaction.Id,
		})
		if err != nil {
			klog.Errorf("list server error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
				TransactionId: transactionResp.Transaction.Id,
			}); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return err
		}

		for _, server := range serversResp.Servers {
			_, err := s.HAProxyClient.DeleteServer(ctx, &haproxyv1.DeleteServerRequest{
				Name:          server.Name,
				BackendName:   backend.Name,
				TransactionId: transactionResp.Transaction.Id,
			})
			if err != nil {
				klog.Errorf("delete server error: %v", err.Error())
				if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
					TransactionId: transactionResp.Transaction.Id,
				}); closeTransactionErr != nil {
					klog.Errorf("close transaction error: %v", err.Error())
				}
				return err
			}
		}

		_, err = s.HAProxyClient.DeleteBackend(ctx, &haproxyv1.DeleteBackendRequest{
			Name:          backend.Name,
			TransactionId: transactionResp.Transaction.Id,
		})
		if err != nil {
			klog.Errorf("delete backend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
				TransactionId: transactionResp.Transaction.Id,
			}); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return err
		}
	}

	if _, err := s.HAProxyClient.CommitTransaction(ctx, &haproxyv1.CommitTransactionRequest{
		TransactionId: transactionResp.Transaction.Id,
	}); err != nil {
		klog.Errorf("commit transaction error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
			TransactionId: transactionResp.Transaction.Id,
		}); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}

		return err
	}

	return nil
}

func (s *ServiceController) EnsureLoadBalancer(ctx context.Context, _ string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	klog.Info("Creating HAProxy LoadBalancer...")
	return s.reconcileLoadBalancer(ctx, service, nodes)
}

func (s *ServiceController) reconcileLoadBalancer(ctx context.Context, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
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

	versionResp, err := s.HAProxyClient.GetVersion(ctx, &haproxyv1.GetVersionRequest{})
	if err != nil {
		klog.Errorf("get current version error: %v", err.Error())
		return nil, err
	}
	transactionResp, err := s.HAProxyClient.CreateTransaction(ctx, &haproxyv1.CreateTransactionRequest{
		Version: versionResp.Version,
	})
	if err != nil {
		klog.Errorf("create transaction error: %v", err.Error())
		return nil, err
	}

	resourcePrefix := fmt.Sprintf("haproxy-%s", service.UID)

	backendsResp, err := s.HAProxyClient.ListBackends(ctx, &haproxyv1.ListBackendsRequest{
		TransactionId: transactionResp.Transaction.Id,
	})
	if err != nil {
		klog.Errorf("list backend error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
			TransactionId: transactionResp.Transaction.Id,
		}); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}
		return nil, err
	}

	// delete all backends and servers
	for _, backend := range backendsResp.Backends {
		if !strings.HasPrefix(backend.Name, resourcePrefix) {
			continue
		}

		serversResp, err := s.HAProxyClient.ListServers(ctx, &haproxyv1.ListServersRequest{
			BackendName:   backend.Name,
			TransactionId: transactionResp.Transaction.Id,
		})
		if err != nil {
			klog.Errorf("list server error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
				TransactionId: transactionResp.Transaction.Id,
			}); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return nil, err
		}

		for _, server := range serversResp.Servers {
			_, err := s.HAProxyClient.DeleteServer(ctx, &haproxyv1.DeleteServerRequest{
				Name:          server.Name,
				BackendName:   backend.Name,
				TransactionId: transactionResp.Transaction.Id,
			})
			if err != nil {
				klog.Errorf("delete server error: %v", err.Error())
				if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
					TransactionId: transactionResp.Transaction.Id,
				}); closeTransactionErr != nil {
					klog.Errorf("close transaction error: %v", err.Error())
				}
				return nil, err
			}
		}

		_, err = s.HAProxyClient.DeleteBackend(ctx, &haproxyv1.DeleteBackendRequest{
			Name:          backend.Name,
			TransactionId: transactionResp.Transaction.Id,
		})
		if err != nil {
			klog.Errorf("delete backend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
				TransactionId: transactionResp.Transaction.Id,
			}); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return nil, err
		}
	}

	frontendsResp, err := s.HAProxyClient.ListFrontends(ctx, &haproxyv1.ListFrontendsRequest{
		TransactionId: transactionResp.Transaction.Id,
	})
	if err != nil {
		klog.Errorf("list frontend error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
			TransactionId: transactionResp.Transaction.Id,
		}); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}
		return nil, err
	}
	for _, frontend := range frontendsResp.Frontends {
		if !strings.HasPrefix(frontend.Name, resourcePrefix) {
			continue
		}

		bindsResp, err := s.HAProxyClient.ListBinds(ctx, &haproxyv1.ListBindsRequest{
			FrontendName:  frontend.Name,
			TransactionId: transactionResp.Transaction.Id,
		})
		if err != nil {
			klog.Errorf("list bind error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
				TransactionId: transactionResp.Transaction.Id,
			}); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return nil, err
		}

		for _, bind := range bindsResp.Binds {
			_, err := s.HAProxyClient.DeleteBind(ctx, &haproxyv1.DeleteBindRequest{
				Name:          bind.Name,
				FrontendName:  frontend.Name,
				TransactionId: transactionResp.Transaction.Id,
			})
			if err != nil {
				klog.Errorf("delete bind error: %v", err.Error())

				if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
					TransactionId: transactionResp.Transaction.Id,
				}); closeTransactionErr != nil {
					klog.Errorf("close transaction error: %v", err.Error())
				}
				return nil, err
			}
		}
		_, err = s.HAProxyClient.DeleteFrontend(ctx, &haproxyv1.DeleteFrontendRequest{
			Name:          frontend.Name,
			TransactionId: transactionResp.Transaction.Id,
		})
		if err != nil {
			klog.Errorf("delete frontend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
				TransactionId: transactionResp.Transaction.Id,
			}); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return nil, err
		}
	}

	// create a new backend and backend servers
	for _, port := range service.Spec.Ports {
		resourceName := fmt.Sprintf("%s-%s-%s", resourcePrefix, port.Name, port.Protocol)
		_, err := s.HAProxyClient.CreateBackend(ctx, &haproxyv1.CreateBackendRequest{
			Backend: &haproxyv1.Backend{
				Name: resourceName,
				Mode: haproxyv1.ProxyMode_PROXY_MODE_TCP,
				Balance: &haproxyv1.BackendBalance{
					Algorithm: haproxyv1.BalanceAlgorithm_BALANCE_ALGORITHM_ROUNDROBIN,
				},
			},
			TransactionId: transactionResp.Transaction.Id,
		})
		if err != nil {
			klog.Errorf("create backend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
				TransactionId: transactionResp.Transaction.Id,
			}); closeTransactionErr != nil {
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
			nodePort := int32(port.NodePort)
			serverName := fmt.Sprintf("server-%s-%s-%d-%d", service.UID, node.Name, port.NodePort, i)
			_, err = s.HAProxyClient.CreateServer(ctx, &haproxyv1.CreateServerRequest{
				Server: &haproxyv1.Server{
					Name:    serverName,
					Address: nodeIp,
					Port:    nodePort,
				},
				BackendName:   resourceName,
				TransactionId: transactionResp.Transaction.Id,
			})
			if err != nil {
				klog.Errorf("create server error: %v", err.Error())
				if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
					TransactionId: transactionResp.Transaction.Id,
				}); closeTransactionErr != nil {
					klog.Errorf("close transaction error: %v", err.Error())
				}
				return nil, err
			}
		}
	}

	// Create new frontend if not exists
	for _, port := range service.Spec.Ports {
		resourceName := fmt.Sprintf("%s-%s-%s", resourcePrefix, port.Name, port.Protocol)
		_, err := s.HAProxyClient.CreateFrontend(ctx, &haproxyv1.CreateFrontendRequest{
			Frontend: &haproxyv1.Frontend{
				Name:           resourceName,
				Mode:           haproxyv1.ProxyMode_PROXY_MODE_TCP,
				DefaultBackend: resourceName,
			},
			TransactionId: transactionResp.Transaction.Id,
		})
		if err != nil {
			klog.Errorf("create frontend error: %v", err.Error())
			if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
				TransactionId: transactionResp.Transaction.Id,
			}); closeTransactionErr != nil {
				klog.Errorf("close transaction error: %v", err.Error())
			}
			return nil, err
		}

		for _, ip := range service.Spec.ExternalIPs {
			bindName := fmt.Sprintf("%s-%s-%s", resourcePrefix, ip, port.Protocol)
			portNum := int32(port.Port)
			_, err := s.HAProxyClient.CreateBind(ctx, &haproxyv1.CreateBindRequest{
				Bind: &haproxyv1.Bind{
					Name:    bindName,
					Address: ip,
					Port:    portNum,
				},
				FrontendName:  resourceName,
				TransactionId: transactionResp.Transaction.Id,
			})
			if err != nil {
				klog.Errorf("create bind error: %v", err.Error())
				if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
					TransactionId: transactionResp.Transaction.Id,
				}); closeTransactionErr != nil {
					klog.Errorf("close transaction error: %v", err.Error())
				}
				return nil, err
			}
		}
	}

	if _, err := s.HAProxyClient.CommitTransaction(ctx, &haproxyv1.CommitTransactionRequest{
		TransactionId: transactionResp.Transaction.Id,
	}); err != nil {
		klog.Errorf("commit transaction error: %v", err.Error())
		if _, closeTransactionErr := s.HAProxyClient.CloseTransaction(ctx, &haproxyv1.CloseTransactionRequest{
			TransactionId: transactionResp.Transaction.Id,
		}); closeTransactionErr != nil {
			klog.Errorf("close transaction error: %v", err.Error())
		}
		return nil, err
	}

	for _, externalIP := range service.Spec.ExternalIPs {
		if externalIP == "" {
			continue
		}
		for _, port := range service.Spec.Ports {
			newStatus.Ingress = append(newStatus.Ingress, v1.LoadBalancerIngress{
				IP: externalIP,
				Ports: []v1.PortStatus{
					{
						Port:     port.Port,
						Protocol: port.Protocol,
					},
				},
			})
		}
	}

	return &newStatus, nil
}
