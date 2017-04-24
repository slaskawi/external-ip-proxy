package kubernetes

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/labels"
	"github.com/slaskawi/external-ip-proxy/logging"
	"os"
)

const (
	ExternalIPsLabelPrefix = "extip"

	Service    = "service"
	Deployment = "deployment"

	ConfigurationServiceLabel = ExternalIPsLabelPrefix + "-controller-" + Service
	ConfigurationServiceName  = ExternalIPsLabelPrefix + "-controller-" + Service

	ProxyServiceLabel = ExternalIPsLabelPrefix + "-proxy-" + Service + "-%v"
	ProxyServiceName  = ExternalIPsLabelPrefix + "-proxy-" + Service + "-%v"

	ProxyDeploymentLabel = ExternalIPsLabelPrefix + "-proxy-" + Deployment + "-%v"
	ProxyDeploymentName  = ExternalIPsLabelPrefix + "-proxy-" + Deployment + "-%v"
)

var Logger *logging.Logger = logging.NewLogger("kubernetes")

type KubeClient struct {
	KubernetesConfigPath string
	kubeClient           *kubernetes.Clientset
}

func NewKubeProxy(KubernetesConfig string) (*KubeClient, error) {
	client := &KubeClient{KubernetesConfigPath: KubernetesConfig}

	config, err := rest.InClusterConfig()
	if err != nil {
		//we are in stanalone mode, let's check if the file exists
		if _, err := os.Stat(KubernetesConfig); err != nil {
			return nil, err
		}
	}

	config, err = clientcmd.BuildConfigFromFlags("", client.KubernetesConfigPath)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	client.kubeClient = clientset

	return client, nil
}

func (client *KubeClient) EnsureServiceIsRunning(ServiceName string, ServiceLabels map[string]string, SourcePorts []int32, DestinationPorts []int32, ExternalIP string) error {
	//Add more ways:
	//https://docs.openshift.org/latest/dev_guide/getting_traffic_into_cluster.html#using-externalIP

	services, err := client.kubeClient.CoreV1().Services("myproject").List(metav1.ListOptions{
		LabelSelector: labels.Set(ServiceLabels).String(),
	})
	if err != nil {
		return err
	}
	numberOfServices := len(services.Items)
	if numberOfServices > 1 {
		return fmt.Errorf("Found more than 1 configuration service with labels %v", ConfigurationServiceLabel)
	} else if numberOfServices == 0 {

		ports := make([]v1.ServicePort, len(SourcePorts))
		for index, sourcePort := range SourcePorts {
			ports[index] = v1.ServicePort{
				Protocol: "TCP",
				Port:     sourcePort,
				TargetPort: intstr.IntOrString{
					IntVal: DestinationPorts[index],
				},
			}
		}

		Logger.Debug("There is no service, creating one")
		service, err := client.kubeClient.CoreV1().Services("myproject").Create(&v1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: "v1",
			},
			Spec: v1.ServiceSpec{
				Type:           "LoadBalancer",
				LoadBalancerIP: ExternalIP,
				Ports:          ports,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   ServiceName,
				Labels: ServiceLabels,
			},
		})
		if err != nil {
			return err
		}
		Logger.Debug("Service created %v", service)
	} else {
		Logger.Debug("The service is fine")
	}

	return nil
}

func (client *KubeClient) EnsurePodIsRunning(PodName string, PodLabels map[string]string, ExposedPorts []int32, Image string, RuntimeParameters []string) error {
	//Add more ways:
	//https://docs.openshift.org/latest/dev_guide/getting_traffic_into_cluster.html#using-externalIP

	pods, err := client.kubeClient.CoreV1().Pods("myproject").List(metav1.ListOptions{
		LabelSelector: labels.Set(PodLabels).String(),
	})
	if err != nil {
		return err
	}

	numberOfPods := len(pods.Items)
	if numberOfPods > 1 {
		return fmt.Errorf("Found more than 1 configuration service with name %v and labels %v", PodName, PodLabels)
	} else if numberOfPods == 0 {

		ports := make([]v1.ContainerPort, len(ExposedPorts))
		for index, sourcePort := range ExposedPorts {
			ports[index] = v1.ContainerPort{
				Protocol:      "TCP",
				ContainerPort: sourcePort,
			}
		}

		Logger.Debug("There is no pod, creating one")
		pod, err := client.kubeClient.CoreV1().Pods("myproject").Create(&v1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			Spec: v1.PodSpec{
				RestartPolicy: "Never",
				Containers: []v1.Container{
					{
						Name:  PodName,
						Image: Image,
						Ports: ports,
						Args: RuntimeParameters,
					},

				},
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   PodName,
				Labels: PodLabels,
			},
		})
		if err != nil {
			return err
		}
		Logger.Debug("Pod [%v] created", pod)
	} else {
		Logger.Debug("The Pod [%v] is fine", PodName)
	}

	return nil
}

func (client *KubeClient) GetPods(PodLabels map[string]string) ([]v1.Pod, error) {
	pods, err := client.kubeClient.CoreV1().Pods("myproject").List(metav1.ListOptions{
		LabelSelector: labels.Set(PodLabels).String(),
	})
	return pods.Items, err
}
