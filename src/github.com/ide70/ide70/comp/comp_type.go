package comp

import "bytes"
import "github.com/ide70/ide70/util/log"
import "io/ioutil"
import "text/template"
import "strings"
import "gopkg.in/yaml.v2"
import "github.com/ide70/ide70/dataxform"

const COMP_PATH = "ide70/comp/"

var logger = log.Logger{"comp"}

// An user defined component
type CompType struct {
	Name string
	Body *template.Template
}

type CompModule struct {
	Body       string
	BodyConsts map[string]interface{}
	Includes   []string
}

func loadCompModule(name string) *CompModule {
	module := &CompModule{}
	logger.Info("loadCompModule", name)
	contentB, err := ioutil.ReadFile(COMP_PATH + name + ".yaml")
	if err != nil {
		logger.Error("Component module ", name, "not found")
		return nil
	}

	decoder := yaml.NewDecoder(bytes.NewReader(contentB))

	var compIf interface{}
	err = decoder.Decode(&compIf)
	if err != nil {
		logger.Error("Component module ", name, "failed to decode:", err.Error())
	}

	var compIfMap map[string]interface{}
	switch compIfT := compIf.(type) {
	case map[interface{}]interface{}:
		compIfMap = dataxform.InterfaceMapToStringMap(compIfT)
	default:
		logger.Error("Component module ", name, "yaml structure is not a map")
		return nil
	}

	module.Body = dataxform.SIMapGetByKeyAsString(compIfMap, "body")
	module.BodyConsts = dataxform.SIMapGetByKeyAsMap(compIfMap, "bodyConsts")
	includes := dataxform.SIMapGetByKeyAsList(compIfMap, "include")
	for _,includeItemIf := range includes {
		module.Includes = append(module.Includes, includeItemIf.(string))
	}

	if module.Body == "" {
		logger.Warning("Component module", name, "has no body")
	}

	return module
}

func parseCompType(name string, appParams *AppParams) *CompType {
	module := loadCompModule(name)
	
	body := ""
	bodyConsts := map[string]interface{}{}
	processedNames := map[string]bool{}
	for _, includeName := range module.Includes {
		if processedNames[includeName] {
			continue
		}
		processedNames[includeName] = true
		include := loadCompModule(includeName)
			body += include.Body
			for k, v := range include.BodyConsts {
				bodyConsts[k] = v
			}
	}

	body += module.Body
	for k, v := range module.BodyConsts {
		bodyConsts[k] = v
	}

	comp := &CompType{}
	comp.Name = name

	var err error
	comp.Body, err = template.New(name).Funcs(template.FuncMap{
		"evalComp": EvalComp,
		"app": func() *AppParams {
			return appParams
		},
		"consts": func() map[string]interface{} {
			return bodyConsts
		},
	}).Parse(body)
	if err != nil {
		logger.Error("Parse Component", err.Error())
		return nil
	}
	CompCache[name] = comp
	logger.Info("parseCompType-end")
	return comp
}

func GetCompType(name string, appParams *AppParams) *CompType {
	compType, has := CompCache[name]
	if has {
		return compType
	}
	return parseCompType(name, appParams)
}

func EvalComp(comp *CompRuntime) string {
	sb := &strings.Builder{}
	comp.CompDef.CompType.Body.Execute(sb, comp.State)
	return sb.String()
}
