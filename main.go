package main

import (
	"flag"
	"github.com/bear-san/haproxy-ccm/controllers"
	haproxyv1 "github.com/bear-san/haproxy-configurator/pkg/haproxy/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"k8s.io/apimachinery/pkg/util/wait"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/cloud-provider/app"
	cloudcontrollerconfig "k8s.io/cloud-provider/app/config"
	"k8s.io/cloud-provider/names"
	"k8s.io/cloud-provider/options"
	"k8s.io/component-base/cli"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
	"os"
)

func main() {
	ccmOptions, err := options.NewCloudControllerManagerOptions()
	if err != nil {
		panic(err)
	}

	haproxyEndpoint := flag.String("haproxy-endpoint", os.Getenv("HAPROXY_ENDPOINT"), "The endpoint of the haproxy gRPC API")
	if *haproxyEndpoint == "" {
		klog.Fatalf("haproxy endpoint is required")
	}

	cloudprovider.RegisterCloudProvider("haproxy", func(config io.Reader) (cloudprovider.Interface, error) {
		// Create gRPC connection
		conn, err := grpc.NewClient(*haproxyEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		
		client := haproxyv1.NewHAProxyManagerServiceClient(conn)
		
		return &controllers.Provider{
			HAProxyClient: client,
			Connection:    conn,
		}, nil
	})

	controllerInitializers := app.DefaultInitFuncConstructors
	controllerAliases := names.CCMControllerAliases()

	fss := cliflag.NamedFlagSets{}

	command := app.NewCloudControllerManagerCommand(ccmOptions, cloudInitializer, controllerInitializers, controllerAliases, fss, wait.NeverStop)
	code := cli.Run(command)
	os.Exit(code)
}

func cloudInitializer(config *cloudcontrollerconfig.CompletedConfig) cloudprovider.Interface {
	cloud, err := cloudprovider.InitCloudProvider("haproxy", config.ComponentConfig.KubeCloudShared.CloudProvider.CloudConfigFile)
	if err != nil {
		klog.Fatalf("Cloud provider could not be initialized: %v", err)
	}
	if cloud == nil {
		klog.Fatalf("Cloud provider is nil")
	}

	if !cloud.HasClusterID() {
		if config.ComponentConfig.KubeCloudShared.AllowUntaggedCloud {
			klog.Warning("detected a cluster without a ClusterID.  A ClusterID will be required in the future.  Please tag your cluster to avoid any future issues")
		} else {
			klog.Fatalf("no ClusterID found.  A ClusterID is required for the cloud provider to function properly.  This check can be bypassed by setting the allow-untagged-cloud option")
		}
	}

	return cloud
}
