package main

import (
	"flag"
	"github.com/bear-san/haproxy-ccm/controllers"
	haproxyv3 "github.com/bear-san/haproxy-go/dataplane/v3"
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

	haproxyBaseUrl := flag.String("haproxy-endpoint", os.Getenv("HAPROXY_ENDPOINT"), "The endpoint of the haproxy API")
	if *haproxyBaseUrl == "" {
		klog.Fatalf("haproxy endpoint is required")
	}
	haproxyAuth := os.Getenv("HAPROXY_AUTH")

	cloudprovider.RegisterCloudProvider("haproxy", func(config io.Reader) (cloudprovider.Interface, error) {
		return &controllers.Provider{
			HAproxyClient: &haproxyv3.Client{
				Credential: haproxyAuth,
				BaseUrl:    *haproxyBaseUrl,
			},
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
