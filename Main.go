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
	kubeConfigLocationFlag = flag.String("c", "", "Kubernetes runtimeConfiguration")
	logLevelFlag = flag.String("loglevel", "Debug", "Log level, e.g. Debug")
	configurationPathFlag = flag.String("runtimeConfiguration", "./configuration.yml", "Path to runtimeConfiguration, e.g. ./runtimeConfiguration.yml")

	logger = logging.NewLogger("main")

	runtimeConfiguration *configuration.Configuration
	kubernetesClient     *kubernetes.KubeClient
	httpServer           *http.HttpServer
)

func main() {
	logger.Info("---- Initialization ----")
	flag.Parse()
	logging.LoggingFromLevel = logging.LogLevel(*logLevelFlag)

	if len(*kubeConfigLocationFlag) == 0 {
		usr, err := user.Current()
		if err != nil {
			logger.Warning("Could not find current user %v", err)
		} else {
			path := fmt.Sprintf("%v/.kube/config", usr.HomeDir)
			kubeConfigLocationFlag = &path
		}
	}

	ConfigurationAsBytes, err := ioutil.ReadFile(*configurationPathFlag)
	if err != nil {
		logger.Error("Could not find configuration, %v", err)
		panic(err)
	}

	runtimeConfiguration, err = configuration.Unmarshal(ConfigurationAsBytes)
	if err != nil {
		logger.Error("Could not initialize configuration, %v", err)
		panic(err)
	}

	httpServer = http.NewHttpServer("0.0.0.0", 8888, runtimeConfiguration)
	httpServer.Start()

	kubernetesClient, err = kubernetes.NewKubeProxy(*kubeConfigLocationFlag, runtimeConfiguration.ExternalIps.Namespace)
	if err != nil {
		logger.Error("Could not initialize Kubernetes client, %v", err)
		panic(err)
	}

	logger.Info("runtimeConfiguration: %v", runtimeConfiguration)
	for {
		logger.Info("---- Getting cluster Pods ----")
		ClusterPods, err := kubernetesClient.GetPods(runtimeConfiguration.Cluster.Labels)

		if runtimeConfiguration.ExternalIps.DynamicIps == false && len(ClusterPods) > len(runtimeConfiguration.ExternalIps.Ips) {
			err = fmt.Errorf("Number of Pods [%v] is greater than number of external IPs [%v]", len(ClusterPods), len(runtimeConfiguration.ExternalIps.Ips))
			logger.Error("%v", err)
			panic(err)
		}

		logger.Info("---- Updating Controller Service ----")
		var ServiceIp string = ""
		if len(runtimeConfiguration.ExternalIps.ServiceIp) > 0 {
			ServiceIp = runtimeConfiguration.ExternalIps.ServiceIp
		}
		Service, err := kubernetesClient.EnsureServiceIsRunning(
			kubernetes.ConfigurationServiceName,
			map[string]string{kubernetes.ExternalIPsLabelPrefix: kubernetes.ConfigurationServiceLabel},
			[]int32{8888},
			[]int32{8888},
			ServiceIp,
			map[string]string{kubernetes.DefaultServiceAppPrefix: kubernetes.DefaultServiceAppName})
		if err != nil {
			logger.Error("%v", err)
		} else {
			if len(Service.Spec.ExternalIPs) > 0 {
				runtimeConfiguration.RuntimeConfiguration.ServiceIp = Service.Spec.ExternalIPs[0]
			}
		}

		logger.Info("---- Adding Marker Labels ----")
		for index, pod := range ClusterPods {
			logger.Debug("Processing Pod %v", pod.Status.PodIP)
			ipForName := fmt.Sprintf("auto-%v", index)
			if !runtimeConfiguration.ExternalIps.DynamicIps {
				ipForName = strings.Replace(runtimeConfiguration.ExternalIps.Ips[index], ".", "-", -1)
			}
			err = kubernetesClient.AddLabelsToPod(
				runtimeConfiguration.Cluster.Labels,
				pod.Name,
				map[string]string{kubernetes.ExternalIPsLabelPrefix: fmt.Sprintf(kubernetes.ProxyPodLabel, ipForName)})
			if err != nil {
				logger.Error("%v", err)
			}
		}

		logger.Info("---- Updating Proxy Services ----")
		runtimeConfiguration.RuntimeConfiguration.ExternalMapping = runtimeConfiguration.RuntimeConfiguration.ExternalMapping[0:0]
		for index, pod := range ClusterPods {
			PodIp := pod.Status.PodIP
			logger.Debug("Processing Pod %v", PodIp)

			ServiceIp := ""
			ipForName := fmt.Sprintf("auto-%v", index)
			if !runtimeConfiguration.ExternalIps.DynamicIps {
				ServiceIp = runtimeConfiguration.ExternalIps.Ips[index]
				ipForName = strings.Replace(ServiceIp, ".", "-", -1)
			}

			PodLabels := map[string]string{kubernetes.ExternalIPsLabelPrefix: fmt.Sprintf(kubernetes.ProxyPodLabel, ipForName)}
			Service, err = kubernetesClient.EnsureServiceIsRunning(
				fmt.Sprintf(kubernetes.ProxyServiceName, ipForName),
				map[string]string{kubernetes.ExternalIPsLabelPrefix: fmt.Sprintf(kubernetes.ProxyServiceLabel, ipForName)},
				runtimeConfiguration.Cluster.Ports,
				runtimeConfiguration.Cluster.Ports,
				ServiceIp,
				PodLabels)
			if err != nil {
				logger.Error("%v", err)
			} else {
				if len(Service.Status.LoadBalancer.Ingress) > 0 {
					Mapping := fmt.Sprintf("%v:%v", PodIp, Service.Status.LoadBalancer.Ingress[0].IP)
					runtimeConfiguration.RuntimeConfiguration.ExternalMapping = append(runtimeConfiguration.RuntimeConfiguration.ExternalMapping, Mapping)
				} else if len(Service.Spec.ExternalIPs) > 0 {
					Mapping := fmt.Sprintf("%v:%v", PodIp, Service.Spec.ExternalIPs[0])
					runtimeConfiguration.RuntimeConfiguration.ExternalMapping = append(runtimeConfiguration.RuntimeConfiguration.ExternalMapping, Mapping)
				}
			}
		}

		logger.Info("---- Removing unnecessary services ----")
		kubernetesClient.RemoveUnnecessaryServices(kubernetes.ExternalIPsLabelPrefix)

		time.Sleep(5 * time.Minute)
	}
}
