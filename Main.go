package main

import (
	"github.com/slaskawi/external-ip-proxy/Proxy"
	"github.com/slaskawi/external-ip-proxy/configuration"
	"github.com/slaskawi/external-ip-proxy/kubernetes"
	"github.com/slaskawi/external-ip-proxy/logging"
	"github.com/slaskawi/external-ip-proxy/http"
	"flag"
	"fmt"
	"time"
	"strings"
)

var (
	localAddr  = flag.String("l", "localhost:1234", "local address")
	remoteAddr = flag.String("r", "ec2-52-215-14-157.eu-west-1.compute.amazonaws.com:8080", "remote address")

	Logger = &logging.Logger{}

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
      - cluster-1
   stateful-set: stateful-set-1
`

func main() {
	Logger.Info("---- Initialization ----")
	flag.Parse()

	Configuration, err = configuration.Unmarshal([]byte(ConfigurationAsString))
	if err != nil {
		Logger.Error("Could not initialize configuration, %v")
		panic(err)
	}

	HTTPServer = http.NewHttpServer("0.0.0.0", 8888, ConfigurationAsString)
	HTTPServer.Start()


	KubernetesClient, err = kubernetes.NewKubeProxy("/home/slaskawi/.kube/config")
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
			8080,
			8080,
			Configuration.ExternalIps.ServiceIp)
		if err != nil {
			Logger.Error("%v", err)
		}

		Logger.Info("---- Updating Proxy Services ----")
		for _, ip := range Configuration.ExternalIps.Ips {
			Logger.Debug("Processing IP %v", ip)

			ipForName := strings.Replace(ip, ".", "-", -1)
			ipAsString := ip

			err = KubernetesClient.EnsureServiceIsRunning(
				fmt.Sprintf(kubernetes.ProxyServiceName, ipForName),
				map[string]string{kubernetes.ExternalIPsLabelPrefix: fmt.Sprintf(kubernetes.ProxyServiceLabel, ipForName)},
				8080,
				8080,
				ipAsString)
			if err != nil {
				Logger.Error("%v", err)
			}
		}
		
		time.Sleep(10 * time.Second)
	}

	fmt.Printf("client %v", KubernetesClient)

	if true {
		return
	}

	p := proxy.NewProxy(*localAddr, *remoteAddr)
	p.Start()

}
