package dataxform

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

func LoadYaml(fileName string) map[string]interface{} {
	contentB, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Println("Yaml file ", fileName, "not found")
		return nil
	}

	decoder := yaml.NewDecoder(bytes.NewReader(contentB))

	var compIf interface{}
	err = decoder.Decode(&compIf)
	if err != nil {
		fmt.Println("Yaml module ", fileName, "failed to decode:", err.Error())
	}

	switch compIfT := compIf.(type) {
	case map[interface{}]interface{}:
		return InterfaceMapToStringMap(compIfT)
	default:
		fmt.Println("Yaml module ", fileName, "yaml structure is not a map")
		return nil
	}
}
