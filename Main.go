package main

import (
	"github.com/slaskawi/external-ip-proxy/configuration"
	"github.com/slaskawi/external-ip-proxy/kubernetes"
	"github.com/slaskawi/external-ip-proxy/logging"
	"github.com/slaskawi/external-ip-proxy/http"
	"flag"
	"fmt"
	"os/user"
	"strings"
	"time"
	"io/ioutil"
)

var (
	KubeConfigLocation = flag.String("c", "", "Kubernetes configuration")

	LogLevel = flag.String("loglevel", "Debug", "Log level, e.g. Debug")

	ConfigurationPath = flag.String("configuration", "./configuration.yml", "Path to configuration, e.g. ./configuration.yml")

	Logger = logging.NewLogger("main")

	Configuration    *configuration.Configuration
	KubernetesClient *kubernetes.KubeClient
	HTTPServer *http.HttpServer
)

func main() {
	Logger.Info("---- Initialization ----")
	flag.Parse()
	logging.LoggingFromLevel = logging.LogLevel(*LogLevel)

	if len(*KubeConfigLocation) == 0 {
		usr, err := user.Current()
		if err != nil {
			Logger.Warning("Could not find current user %v", err)
		} else {
			path := fmt.Sprintf("%v/.kube/config", usr.HomeDir)
			KubeConfigLocation = &path
		}
	}

	ConfigurationAsBytes, err := ioutil.ReadFile(*ConfigurationPath)
	if err != nil {
		Logger.Error("Could not find configuration, %v", err)
		panic(err)
	}

	Configuration, err = configuration.Unmarshal(ConfigurationAsBytes)
	if err != nil {
		Logger.Error("Could not initialize configuration, %v", err)
		panic(err)
	}
	ConfigurationAsString := string(ConfigurationAsBytes)
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
			Configuration.ExternalIps.ServiceIp,
			nil)
		if err != nil {
			Logger.Error("%v", err)
		}

		Logger.Info("---- Adding Marker Labels ----")
		for index, pod := range ClusterPods {
			Logger.Debug("Processing Pod %v", pod.Status.PodIP)
			ip := Configuration.ExternalIps.Ips[index]
			ipForName := strings.Replace(ip, ".", "-", -1)
			err = KubernetesClient.AddLabelsToPod(
				Configuration.Cluster.Labels,
				pod.Name,
				map[string]string{kubernetes.ExternalIPsLabelPrefix: fmt.Sprintf(kubernetes.ProxyServiceLabel, ipForName)})
			if err != nil {
				Logger.Error("%v", err)
			}
		}

		Logger.Info("---- Updating Proxy Services ----")
		for index := range ClusterPods {
			Ip := Configuration.ExternalIps.Ips[index]
			Logger.Debug("Processing IP %v", Ip)

			ipForName := strings.Replace(Ip, ".", "-", -1)
			ipAsString := Ip
			PodLabels := map[string]string{kubernetes.ExternalIPsLabelPrefix: fmt.Sprintf(kubernetes.ProxyServiceLabel, ipForName)}

			err = KubernetesClient.EnsureServiceIsRunning(
				fmt.Sprintf(kubernetes.ProxyServiceName, ipForName),
				map[string]string{kubernetes.ExternalIPsLabelPrefix: fmt.Sprintf(kubernetes.ProxyServiceLabel, ipForName)},
				Configuration.Cluster.Ports,
				Configuration.Cluster.Ports,
				ipAsString,
				PodLabels)
			if err != nil {
				Logger.Error("%v", err)
			}
		}

		Logger.Info("---- Removing unnecessary services ----")
		KubernetesClient.RemoveUnnecessaryServices(kubernetes.ExternalIPsLabelPrefix)

		time.Sleep(10 * time.Second)
	}

}