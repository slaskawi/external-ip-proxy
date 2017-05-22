package main

import (
	"github.com/slaskawi/external-ip-proxy/configuration"
	"github.com/slaskawi/external-ip-proxy/kubernetes"
	"github.com/slaskawi/external-ip-proxy/logging"
	"github.com/slaskawi/external-ip-proxy/http"
	"flag"
	"fmt"
	"os/user"
	"time"
	"io/ioutil"
	"strings"
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

	HTTPServer = http.NewHttpServer("0.0.0.0", 8888, Configuration)
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

		if Configuration.ExternalIps.DynamicIps == false && len(ClusterPods) > len(Configuration.ExternalIps.Ips) {
			err = fmt.Errorf("Number of Pods [%v] is greater than number of external IPs [%v]", len(ClusterPods), len(Configuration.ExternalIps.Ips))
			Logger.Error("%v", err)
			panic(err)
		}

		Logger.Info("---- Updating Controller Service ----")
		var ServiceIp string = ""
		if !Configuration.ExternalIps.DynamicIps {
			ServiceIp = Configuration.ExternalIps.ServiceIp
		}
		Service, err := KubernetesClient.EnsureServiceIsRunning(
			kubernetes.ConfigurationServiceName,
			map[string]string{kubernetes.ExternalIPsLabelPrefix: kubernetes.ConfigurationServiceLabel},
			[]int32{8080},
			[]int32{8080},
			ServiceIp,
			nil)
		if err != nil {
			Logger.Error("%v", err)
		} else {
			if len(Service.Spec.ExternalIPs) > 0 {
				Configuration.RuntimeConfiguration.ServiceIp = Service.Spec.ExternalIPs[0]
			}
		}

		Logger.Info("---- Adding Marker Labels ----")
		for index, pod := range ClusterPods {
			Logger.Debug("Processing Pod %v", pod.Status.PodIP)
			ipForName := fmt.Sprintf("auto-%v", index)
			if !Configuration.ExternalIps.DynamicIps {
				ipForName = strings.Replace(Configuration.ExternalIps.Ips[index], ".", "-", -1)
			}
			err = KubernetesClient.AddLabelsToPod(
				Configuration.Cluster.Labels,
				pod.Name,
				map[string]string{kubernetes.ExternalIPsLabelPrefix: fmt.Sprintf(kubernetes.ProxyPodLabel, ipForName)})
			if err != nil {
				Logger.Error("%v", err)
			}
		}

		Logger.Info("---- Updating Proxy Services ----")
		Configuration.RuntimeConfiguration.ExternalMapping = Configuration.RuntimeConfiguration.ExternalMapping[0:0]
		for index, pod := range ClusterPods {
			PodIp := pod.Status.PodIP
			Logger.Debug("Processing Pod %v", PodIp)

			ServiceIp := ""
			ipForName := fmt.Sprintf("auto-%v", index)
			if !Configuration.ExternalIps.DynamicIps {
				ServiceIp = Configuration.ExternalIps.Ips[index]
				ipForName = strings.Replace(ServiceIp, ".", "-", -1)
			}

			PodLabels := map[string]string{kubernetes.ExternalIPsLabelPrefix: fmt.Sprintf(kubernetes.ProxyPodLabel, ipForName)}
			Service, err = KubernetesClient.EnsureServiceIsRunning(
				fmt.Sprintf(kubernetes.ProxyServiceName, ipForName),
				map[string]string{kubernetes.ExternalIPsLabelPrefix: fmt.Sprintf(kubernetes.ProxyServiceLabel, ipForName)},
				Configuration.Cluster.Ports,
				Configuration.Cluster.Ports,
				ServiceIp,
				PodLabels)
			if err != nil {
				Logger.Error("%v", err)
			} else {
				if len(Service.Spec.ExternalIPs) > 0 {
					Mapping := fmt.Sprintf("%v:%v", PodIp, Service.Spec.ExternalIPs[0])
					Configuration.RuntimeConfiguration.ExternalMapping = append(Configuration.RuntimeConfiguration.ExternalMapping, Mapping)
				}
			}
		}

		Logger.Info("---- Removing unnecessary services ----")
		KubernetesClient.RemoveUnnecessaryServices(kubernetes.ExternalIPsLabelPrefix)

		time.Sleep(10 * time.Second)
	}

}
