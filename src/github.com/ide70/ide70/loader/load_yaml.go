package loader

import (
	"bytes"
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/util/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
	"text/template"
)

var logger = log.Logger{"loader"}

var dynConfigCache = map[string]*TemplatedYaml{}

const dcfgPath = "ide70/dcfg/"

type TemplatedYaml struct {
	Def map[string]interface{}
}

func GetTemplatedYaml(name string) *TemplatedYaml {
	yamlData := dynConfigCache[name]
	if yamlData == nil {
		yamlData = LoadTemplatedYaml(name)
		dynConfigCache[name] = yamlData
	}
	return yamlData
}

func DropTemplatedYaml(name string) {
	logger.Info("drop templatedYaml", name)
	delete(dynConfigCache, name)
}

func LoadTemplatedYaml(name string) *TemplatedYaml {
	module := &TemplatedYaml{}
	logger.Info("loadTemplatedYaml", name)
	contentB, err := ioutil.ReadFile(dcfgPath + name + ".yaml")
	if err != nil {
		logger.Error("Yaml module ", name, "not found")
		return nil
	}

	decoder := yaml.NewDecoder(bytes.NewReader(contentB))

	var compIf interface{}
	err = decoder.Decode(&compIf)
	if err != nil {
		logger.Error("Yaml module ", name, "failed to decode:", err.Error())
	}

	switch compIfT := compIf.(type) {
	case map[interface{}]interface{}:
		module.Def = dataxform.InterfaceMapToStringMap(compIfT)
	default:
		logger.Error("Yaml module ", name, "yaml structure is not a map")
		return nil
	}
	dataxform.SIMapApplyFn(module.Def, func(entry dataxform.CollectionEntry) {
		switch vT := entry.Value().(type) {
		case string:
			if strings.HasPrefix(vT, "TEMPLATE ") {
				var err error
				template, err := template.New(name).Parse(strings.TrimPrefix(vT, "TEMPLATE "))
				if err != nil {
					logger.Error("Parse Yaml module template", entry.LinearKey(), err.Error())
				} else {
					entry.Update(template)
				}
			}
		}
	})
	logger.Info("loaded: ", module.Def)

	return module
}
