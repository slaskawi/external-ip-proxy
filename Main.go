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
	ProxyLocalAddress  = flag.String("l", "", "An address for serving the proxy, e.g. localhost:8080")
	ProxyRemoteAddress = flag.String("r", "", "A remote address, e.g. google.com")

	KubeConfigLocation = flag.String("c", "", "Kubernetes configuration")

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
      app: infinispan-server
   ports:
      - 11222
#   stateful-set: stateful-set-1
`

func main() {
	Logger.Info("---- Initialization ----")
	flag.Parse()

	IsInMasterMode := true

	if len(*ProxyLocalAddress) != 0 && len(*ProxyRemoteAddress) != 0 {
		IsInMasterMode = false
	}

	if IsInMasterMode {
		if len(*KubeConfigLocation) == 0 {
			usr, err := user.Current()
			if err != nil {
				Logger.Warning("Could not find current user %v", err)
			} else {
				path := fmt.Sprintf("%v/.kube/config", usr.HomeDir)
				KubeConfigLocation = &path
			}
		}

		Configuration, err = configuration.Unmarshal([]byte(ConfigurationAsString))
		if err != nil {
			Logger.Error("Could not initialize configuration, %v", err)
			panic(err)
		}

		HTTPServer = http.NewHttpServer("0.0.0.0", 8888, ConfigurationAsString)
		HTTPServer.Start()


		KubernetesClient, err = kubernetes.NewKubeProxy(*KubeConfigLocation)
		if err != nil {
			Logger.Error("Could not initialize Kubernetes client, %v", err)
			panic(err)
		}

		Logger.Info("Configuration: %v", Configuration)
		for {
			Logger.Info("---- Getting cluster Pods ----")
			ClusterPods, err := KubernetesClient.GetPods(Configuration.Cluster.Labels)

			if len(ClusterPods) > len(Configuration.ExternalIps.Ips) {
				err = fmt.Errorf("Number of Pods [%v] is greater than number of external IPs [%v]", len(ClusterPods), len(Configuration.ExternalIps.Ips))
				Logger.Error("%v", err)
				panic(err)
			}

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

			Logger.Info("---- Updating Proxy Services ----")
			for index := range ClusterPods {
				Ip := Configuration.ExternalIps.Ips[index]
				Logger.Debug("Processing IP %v", Ip)

				ipForName := strings.Replace(Ip, ".", "-", -1)
				ipAsString := Ip

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

			Logger.Info("---- Updating Proxy Deployment ----")
			for index, Pod := range ClusterPods {
				Ip := Configuration.ExternalIps.Ips[index]
				Logger.Debug("Processing Deployment for IP %v", Ip)

				SanitizedIP := strings.Replace(Ip, ".", "-", -1)
				PodName := fmt.Sprintf(kubernetes.ProxyDeploymentName, SanitizedIP)
				PodLabels := map[string]string{kubernetes.ExternalIPsLabelPrefix: fmt.Sprintf(kubernetes.ProxyDeploymentLabel, SanitizedIP)}

				ProxyFromIP := Pod.Status.PodIP
				ProxyToIP := "localhost"

				var RuntimeParameters = []string{
					"go run Main.go",
					fmt.Sprintf("-r %v:%v", ProxyFromIP, Configuration.Cluster.Ports[0]),
					fmt.Sprintf("-l %v:%v", ProxyToIP, Configuration.Cluster.Ports[0]),
				}

				err = KubernetesClient.EnsurePodIsRunning(
					PodName,
					PodLabels,
					[]int32{8080},
					"docker.io/slaskawi/external-ip-proxy",
					RuntimeParameters)
				if err != nil {
					Logger.Error("%v", err)
				}
			}

			time.Sleep(20 * time.Second)
		}
	} else {
		Logger.Info("---- Slave mode ----")
		p := proxy.NewProxy(*ProxyLocalAddress, *ProxyRemoteAddress)
		err = p.Start()
		if err != nil {
			Logger.Error("Proxy error %v", err)
			panic(err)
		}
	}

}
