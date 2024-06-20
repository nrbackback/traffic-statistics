package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type yamlParser struct{}

func (yp *yamlParser) parse(filepath string) (map[string]interface{}, error) {
	configFile, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	f, _ := configFile.Stat()
	if f.Size() == 0 {
		return nil, fmt.Errorf("config file (%s) is empty", filepath)
	}
	buffer := make([]byte, f.Size())
	_, err = configFile.Read(buffer)
	if err != nil {
		return nil, err
	}
	config := make(map[string]interface{})
	err = yaml.Unmarshal(buffer, &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
