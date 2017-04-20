package main

import (
	"github.com/slaskawi/external-ip-proxy/Proxy"
	"github.com/slaskawi/external-ip-proxy/configuration"
	"github.com/slaskawi/external-ip-proxy/kubernetes"
	"github.com/slaskawi/external-ip-proxy/logging"
	"github.com/slaskawi/external-ip-proxy/http"
	"flag"
	"fmt"
	"os/user"
	"strings"
	"time"
)

var (
	localAddr  = flag.String("l", "localhost:1234", "local address")
	remoteAddr = flag.String("r", "ec2-52-215-14-157.eu-west-1.compute.amazonaws.com:8080", "remote address")

	kubeConfigLocation = flag.String("c", "", "Kubernetes configuration")

	masterLocation = flag.String("m", "", "Master service location")

	Logger = logging.NewLogger("main")

	Configuration    *configuration.Configuration
	KubernetesClient *kubernetes.KubeClient
	HTTPServer *http.HttpServer

	err error
)

var ConfigurationAsString = `
---
# A full configuration used for testing
external-ips:
   service-ip: 172.29.0.1
   ips:
      - 172.29.0.2
      - 172.29.0.3
      - 172.29.0.4
#   ip-range:
#      from: 127.0.0.1/16
#      to: 127.0.0.1/16
cluster:
   labels:
      - app=infinispan-server
   ports:
      - 8080
#   stateful-set: stateful-set-1
`

func main() {
	Logger.Info("---- Initialization ----")
	flag.Parse()

	IsInMasterMode := true

	if len(*masterLocation) != 0 {
		//are we in slave (proxy-only) mode?
		IsInMasterMode = false
	}

	if IsInMasterMode {
		if len(*kubeConfigLocation) == 0 {
			usr, err := user.Current()
			if err != nil {
				Logger.Warning("Could not find current user %v", err)
			} else {
				path := fmt.Sprintf("%v/.kube/config", usr.HomeDir)
				kubeConfigLocation = &path
			}
		}

		Configuration, err = configuration.Unmarshal([]byte(ConfigurationAsString))
		if err != nil {
			Logger.Error("Could not initialize configuration, %v", err)
			panic(err)
		}

		HTTPServer = http.NewHttpServer("0.0.0.0", 8888, ConfigurationAsString)
		HTTPServer.Start()


		KubernetesClient, err = kubernetes.NewKubeProxy(*kubeConfigLocation)
		if err != nil {
			Logger.Error("Could not initialize Kubernetes client, %v", err)
			panic(err)
		}

		Logger.Info("Configuration: %v", Configuration)
		for {
			Logger.Info("---- Updating Controller Service ----")
			err = KubernetesClient.EnsureServiceIsRunning(
				kubernetes.ConfigurationServiceName,
				map[string]string{kubernetes.ExternalIPsLabelPrefix: kubernetes.ConfigurationServiceLabel},
				[]int32{8080},
				[]int32{8080},
				Configuration.ExternalIps.ServiceIp)
			if err != nil {
				Logger.Error("%v", err)
			}

			Logger.Info("---- Updating Proxy Deployment ----")
			for _, ip := range Configuration.ExternalIps.Ips {
				Logger.Debug("Processing Deployment for IP %v", ip)

				ipForName := strings.Replace(ip, ".", "-", -1)

				err = KubernetesClient.EnsurePodIsRunning(
					fmt.Sprintf(kubernetes.ProxyDeploymentName, ipForName),
					map[string]string{kubernetes.ExternalIPsLabelPrefix: fmt.Sprintf(kubernetes.ProxyDeploymentLabel, ipForName)},
					[]int32{8080},
					"slaskawi/external-ip-proxy")
				if err != nil {
					Logger.Error("%v", err)
				}
			}

			Logger.Info("---- Updating Proxy Services ----")
			for _, ip := range Configuration.ExternalIps.Ips {
				Logger.Debug("Processing IP %v", ip)

				ipForName := strings.Replace(ip, ".", "-", -1)
				ipAsString := ip

				err = KubernetesClient.EnsureServiceIsRunning(
					fmt.Sprintf(kubernetes.ProxyServiceName, ipForName),
					map[string]string{kubernetes.ExternalIPsLabelPrefix: fmt.Sprintf(kubernetes.ProxyServiceLabel, ipForName)},
					Configuration.Cluster.Ports,
					Configuration.Cluster.Ports,
					ipAsString)
				if err != nil {
					Logger.Error("%v", err)
				}
			}

			time.Sleep(20 * time.Second)
		}
	} else {
		//slave only mode
		Logger.Info("Operating in slave mode, master address [%v]", masterLocation)

	}



	fmt.Printf("client %v", KubernetesClient)

	if true {
		return
	}

	p := proxy.NewProxy(*localAddr, *remoteAddr)
	p.Start()

}
