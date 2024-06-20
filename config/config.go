package config

import (
	"errors"
	"strings"
)

type Parser interface {
	parse(filename string) (map[string]interface{}, error)
}

func ParseConfig(filename string) (map[string]interface{}, error) {
	lowerFilename := strings.ToLower(filename)
	if strings.HasSuffix(lowerFilename, ".yaml") || strings.HasSuffix(lowerFilename, ".yml") {
		yp := &yamlParser{}
		return yp.parse(filename)
	}
	return nil, errors.New("unknown config format. config filename should ends with yaml|yml")
}
