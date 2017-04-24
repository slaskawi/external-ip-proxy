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
		Labels map[string]string `yaml:"labels"`
		Ports [] int32 `yaml:"ports"`
		StatefulSet string `yaml:"stateful-set"`
	} `yaml:"cluster"`
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