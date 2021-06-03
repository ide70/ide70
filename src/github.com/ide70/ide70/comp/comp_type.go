package comp

import "fmt"
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
	Name          string
	Body          *template.Template
	EventsHandler *CompDefEventsHandler
	AccessibleDef map[string]interface{}
}

type CompModule struct {
	Body       string
	BodyConsts map[string]interface{}
	Includes   []string
	Def        map[string]interface{}
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
	for _, includeItemIf := range includes {
		module.Includes = append(module.Includes, includeItemIf.(string))
	}
	module.Def = compIfMap

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
	comp.EventsHandler = ParseEventHandlers(module.Def, nil, nil, nil)
	comp.AccessibleDef = map[string]interface{}{}

	// TODO: list of non-accessible definitions
	comp.AccessibleDef["eventHandlers"] = module.Def["eventHandlers"]
	comp.AccessibleDef["autoInclude"] = module.Def["autoInclude"]
	comp.AccessibleDef["css"] = module.Def["css"]
	comp.AccessibleDef["injectRootComp"] = module.Def["injectRootComp"]
	comp.AccessibleDef["injectToComp"] = module.Def["injectToComp"]

	var err error
	comp.Body, err = template.New(name).Funcs(template.FuncMap{
		"evalComp":     EvalComp,
		"generateComp": GenerateComp,
		"eventHandler": GenerateEventHandler,
		"eventHandlerWithKey": GenerateEventHandlerWithKey,
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
	comp.Render(sb)
	return sb.String()
}

func GenerateEventHandler(comp *CompRuntime, eventTypeCli, eventTypeSvr string) string {
	if eventTypeSvr == "" {
		eventTypeSvr = eventTypeCli
	}
	return fmt.Sprintf(" %s=\"se(event,'%s',%d,null)\"", eventTypeCli, eventTypeSvr, comp.Sid())
}

func GenerateEventHandlerWithKey(comp *CompRuntime, eventTypeCli, key string) string {
	return fmt.Sprintf(" %s=\"se(event,'%s',%d,'%s')\"", eventTypeCli, eventTypeCli, comp.Sid(), key)
}

func GenerateComp(parentComp *CompRuntime, sourceChildRef string, genRuntimeRefIf interface{}, context interface{}) string {
	genRuntimeRef := dataxform.IAsString(genRuntimeRefIf)
	genChildRefId := fmt.Sprintf("%s.%s_%s", parentComp.ChildRefId(), sourceChildRef, genRuntimeRef)
	comp := parentComp.GenChilden[genChildRefId]
	if comp == nil {
		logger.Info("genRuntimeRef", genChildRefId)
		srcCompDef := parentComp.Unit.UnitDef.CompsMap[sourceChildRef]
		if srcCompDef == nil {
			logger.Warning("source component not found:", sourceChildRef)
			return ""
		}
		comp = parentComp.Unit.InstantiateComp(srcCompDef, genChildRefId)
		comp.State["parentContext"] = context
		parentComp.GenChilden[genChildRefId] = comp
	}
	sb := &strings.Builder{}
	comp.Render(sb)
	return sb.String()
}
