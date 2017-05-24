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
	"reflect"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const (
	ExternalIPsLabelPrefix = "extip"

	Service    = "service"
	Pod	   = "pod"

	ConfigurationServiceLabel = ExternalIPsLabelPrefix + "-controller-" + Service
	ConfigurationServiceName  = ExternalIPsLabelPrefix + "-controller-" + Service

	ProxyServiceLabel = ExternalIPsLabelPrefix + "-proxy-" + Service + "-%v"
	ProxyServiceName  = ExternalIPsLabelPrefix + "-proxy-" + Service + "-%v"

	ProxyPodLabel = ExternalIPsLabelPrefix + "-pod-" + "%v"
	ProxyPodName  = ExternalIPsLabelPrefix + "-pod-" + "%v"

)

var Logger *logging.Logger = logging.NewLogger("kubernetes")

type KubeClient struct {
	KubernetesConfigPath string
	kubeClient           *kubernetes.Clientset
	Namespace            string
}

func NewKubeProxy(KubernetesConfig string, Namespace string) (*KubeClient, error) {
	client := &KubeClient{KubernetesConfigPath: KubernetesConfig, Namespace: Namespace}

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

func (client *KubeClient) AddLabelsToPod(labels map[string]string, podName string, labelsToBeAdded map[string]string) error {
	pods, err := client.GetPods(labels)
	if err != nil {
		return err
	}
	for _, pod := range pods {
		if podName == pod.Name {
			containsAllLabels := true
			for label := range labelsToBeAdded {
				if _, ok := pod.ObjectMeta.Labels[label]; !ok {
					containsAllLabels = false
					Logger.Debug("Pod [%v] will be updated with additional labels", pod)
					break
				}
			}
			if !containsAllLabels {
				allLabels := pod.ObjectMeta.Labels
				mapCopy(allLabels, labelsToBeAdded)
				Logger.Debug("New labels for Pod [%v] [%v]", pod, allLabels)
				pod.ObjectMeta.Labels = allLabels
				_, err = client.kubeClient.CoreV1().Pods(client.Namespace).Update(&pod)
				if err != nil {
					return err
				}

			}
		}
	}
	return nil
}

func (client *KubeClient) RemoveUnnecessaryServices(serviceLabels string) error {
	services, err := client.kubeClient.CoreV1().Services(client.Namespace).List(metav1.ListOptions{
		LabelSelector: serviceLabels,
	})
	if err != nil {
		return err
	}

	for _, service := range services.Items {
		selector := service.Spec.Selector
		pods, err := client.kubeClient.CoreV1().Pods(client.Namespace).List(metav1.ListOptions{
			LabelSelector: labels.Set(selector).String(),
		})
		if err != nil {
			return err
		}
		if len(pods.Items) == 0 {
			Logger.Debug("Service [%v] does not have Pod attached. Removing", service)
			err := client.kubeClient.CoreV1().Services(client.Namespace).Delete(service.Name, &metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (client *KubeClient) EnsureServiceIsRunning(ServiceName string, ServiceLabels map[string]string, SourcePorts []int32, DestinationPorts []int32, ExternalIP string, Selector map[string]string) (*v1.Service, error) {
	//Add more ways:
	//https://docs.openshift.org/latest/dev_guide/getting_traffic_into_cluster.html#using-externalIP

	services, err := client.kubeClient.CoreV1().Services(client.Namespace).List(metav1.ListOptions{
		LabelSelector: labels.Set(ServiceLabels).String(),
	})
	if err != nil {
		return nil, err
	}
	numberOfServices := len(services.Items)
	if numberOfServices > 1 {
		return nil, fmt.Errorf("Found more than 1 configuration service with labels %v", ConfigurationServiceLabel)
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
		service, err := client.kubeClient.CoreV1().Services(client.Namespace).Create(&v1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: "v1",
			},
			Spec: v1.ServiceSpec{
				Type:           "LoadBalancer",
				LoadBalancerIP: ExternalIP,
				Ports:          ports,
				Selector:       Selector,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   ServiceName,
				Labels: ServiceLabels,
			},
		})
		return service, err

	} else {
		Logger.Debug("The service is fine")
	}

	return &services.Items[0], nil
}

func (client *KubeClient) EnsurePodIsRunning(PodName string, PodLabels map[string]string, ExposedPorts []int32, Image string, Command []string, RuntimeParameters []string) error {
	//Add more ways:
	//https://docs.openshift.org/latest/dev_guide/getting_traffic_into_cluster.html#using-externalIP

	pods, err := client.kubeClient.CoreV1().Pods(client.Namespace).List(metav1.ListOptions{
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
		pod, err := client.kubeClient.CoreV1().Pods(client.Namespace).Create(&v1.Pod{
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
						Command: Command,
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
	pods, err := client.kubeClient.CoreV1().Pods(client.Namespace).List(metav1.ListOptions{
		LabelSelector: labels.Set(PodLabels).String(),
	})
	return pods.Items, err
}

func mapCopy(dst, src interface{}) {
	dv, sv := reflect.ValueOf(dst), reflect.ValueOf(src)

	for _, k := range sv.MapKeys() {
		dv.SetMapIndex(k, sv.MapIndex(k))
	}
}
