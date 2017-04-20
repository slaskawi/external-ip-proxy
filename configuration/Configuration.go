package configuration

import (
	"fmt"
	"gopkg.in/yaml.v2"
)

type Configuration struct {
	ExternalIps struct {
		ServiceIp string `yaml:"service-ip"`
		Ips [] string `yaml:"ips"`
		IpRange struct {
			From string `yaml:"from"`
			To string `yaml:"to"`
		} `yaml:"ip-range"`
	} `yaml:"external-ips"`
	Cluster struct {
		Labels [] string `yaml:"labels"`
		Ports [] int32 `yaml:"ports"`
		StatefulSet string `yaml:"stateful-set"`
	} `yaml:"cluster"`
}


var data = `
---
# A full configuration used for testing
external-ips:
   service-ip: 127.0.0.2
   ips:
      - 127.0.0.1
      - 192.168.0.1
   ip-range:
      from: 127.0.0.1/16
      to: 127.0.0.1/16
cluster:
   labels:
      - cluster-1
   ports:
      - 8080
   stateful-set: stateful-set-1
`

func main() {
	t, _ := Unmarshal([]byte(data))
	fmt.Print("%v", t)
}

func Unmarshal(data []byte) (*Configuration, error) {
	parsedConfiguration := &Configuration{}
	err := yaml.Unmarshal(data, parsedConfiguration)
	if err != nil {
		return nil, fmt.Errorf("Could not unmarshal yaml %v", err)
	}
	return parsedConfiguration, nil
}

func Marshal(configuration *Configuration) (string, error) {
	ret, err := yaml.Marshal(configuration)
	if err != nil {
		return "", err
	}
	return string(ret), nil

}