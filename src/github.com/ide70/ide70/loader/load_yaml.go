package loader

import (
	"bytes"
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/util/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"reflect"
	"strings"
	"text/template"
)

var logger = log.Logger{"loader"}

var dynConfigCache = map[string]*TemplatedYaml{}

const dcfgPath = "ide70/dcfg/"

type TemplatedYaml struct {
	Def  map[string]interface{}
	Defs []interface{}
	IDef interface{}
}

func GetTemplatedYaml(name string, basePath string) *TemplatedYaml {
	defer func() {
        rc := recover()
        if (rc != nil) {
            logger.Error("GetTemplatedYaml panic:", rc)
            return
        }
    }()
	
	if basePath == "" {
		basePath = dcfgPath
	}
	yamlData := dynConfigCache[basePath+name]
	if yamlData == nil {
		logger.Info("NO CACHE:" + basePath + name)
		yamlData = LoadTemplatedYaml(name, basePath)
		dynConfigCache[basePath+name] = yamlData
	}
	return yamlData
}

func DropTemplatedYaml(name string) {
	logger.Info("drop templatedYaml", name)
	delete(dynConfigCache, name)
}

func LoadTemplatedYaml(name, basePath string) *TemplatedYaml {
	logger.Info("loadTemplatedYaml", name)
	contentB, err := ioutil.ReadFile(basePath + name + ".yaml")
	if err != nil {
		logger.Error("Yaml module ", name, "at", basePath, "not found")
		return nil
	}
	return ConvertTemplatedYaml(contentB, name)
}

func ConvertTemplatedYaml(contentB []byte, name string) *TemplatedYaml {
	module := &TemplatedYaml{}
	decoder := yaml.NewDecoder(bytes.NewReader(contentB))

	var compIf interface{}
	err := decoder.Decode(&compIf)
	if err != nil {
		logger.Error("Yaml module ", name, "failed to decode:", err.Error())
		return nil
	}

	switch compIfT := compIf.(type) {
	case map[interface{}]interface{}:
		module.Def = dataxform.InterfaceMapToStringMap(compIfT)
		module.IDef = module.Def
	case []interface{}:
		module.Defs = dataxform.InterfaceListReplaceMapKeyToString(compIfT)
		module.IDef = module.Defs
	default:
		logger.Error("Yaml module ", name, "yaml structure is not a map, but:", reflect.TypeOf(compIf))
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
	logger.Info("converted: ", module.Def)

	return module
}
