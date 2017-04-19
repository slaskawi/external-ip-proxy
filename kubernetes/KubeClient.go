package kubernetes

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/rest"
	"time"
	"k8s.io/apimachinery/pkg/labels"
	"github.com/slaskawi/external-ip-proxy/logging"
	"os"
)

const (
	ExternalIPsLabelPrefix = "extip"

	Service = "service"

	ConfigurationServiceLabel = ExternalIPsLabelPrefix + "-controller-" + Service
	ConfigurationServiceName  = ExternalIPsLabelPrefix + "-controller-" + Service

	ProxyServiceLabel = ExternalIPsLabelPrefix + "-proxy-" + Service + "-%v"
	ProxyServiceName  = ExternalIPsLabelPrefix + "-proxy-" + Service + "-%v"
)

var Logger *logging.Logger = &logging.Logger{}

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

func (client *KubeClient) EnsureServiceIsRunning(ServiceName string, ServiceLabels map[string]string, SourcePort int32, DestinationPort int32, ExternalIP string) error {
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
		Logger.Debug("There is no service, creating one")
		service, err := client.kubeClient.CoreV1().Services("myproject").Create(&v1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: "v1",
			},
			Spec: v1.ServiceSpec{
				Type: "LoadBalancer",
				LoadBalancerIP: ExternalIP,
				Ports: []v1.ServicePort{
					{
						Protocol: "TCP",
						Port:     SourcePort,
						TargetPort: intstr.IntOrString{
							IntVal: DestinationPort,
						},
					},
				},
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


func main() {
	//kubeconfig := flag.String("kubeconfig", "/home/slaskawi/.kube/config", "absolute path to the kubeconfig file")
	//flag.Parse()
	// uses the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", "/home/slaskawi/.kube/config")
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	for {
		pods, err := clientset.Core().Pods("myproject").List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
		time.Sleep(10 * time.Second)
	}
}
