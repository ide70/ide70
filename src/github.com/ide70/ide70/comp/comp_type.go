package comp

import "fmt"
import "bytes"
import "github.com/ide70/ide70/util/log"
import "io/ioutil"
import "text/template"
import "strings"
import "gopkg.in/yaml.v2"
import "github.com/ide70/ide70/dataxform"
import "regexp"

const COMP_PATH = "ide70/comp/"

var logger = log.Logger{"comp"}
var reSubComp = regexp.MustCompile(`<(\w+)[^<>]+\{\{\.sid\}\}(-\w+)`)

// An user defined component
type CompType struct {
	Name          string
	Body          *template.Template
	SubBodies     map[string]*template.Template
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

func extractSubcomponents(comp *CompType, body string, appParams *AppParams) {
	subs := reSubComp.FindAllStringSubmatch(body, -1)
	subIdxs := reSubComp.FindAllStringSubmatchIndex(body, -1)
	for i, sub := range subs {
		tagName := sub[1]
		subCompName := sub[2]
		subIdx := subIdxs[i]
		matchIdx := subIdx[0]
		subCompStr := cutToClosingTag(body[matchIdx:], tagName)
		logger.Info("subcomp:", subCompName, subCompStr)
		comp.SubBodies[subCompName] = createTemplate(subCompStr, comp.Name + subCompName, appParams, nil)
	}
}

func cutToClosingTag(s, tagName string) string {
	r := regexp.MustCompile(fmt.Sprintf(`<%s|<\/%s>`, tagName, tagName))
	subs := r.FindAllStringSubmatch(s, -1)
	subIdxs := r.FindAllStringSubmatchIndex(s, -1)
	openTags := 0
	for i, sub := range subs {
		if strings.HasPrefix(sub[0], "</") {
			openTags--;
		} else {
			openTags++;
		}
		if openTags == 0 {
			return s[:subIdxs[i][0]+len(sub[0])]
		}
	}
	return ""
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

	comp.Body = createTemplate(body, name, appParams, bodyConsts)
	if comp.Body == nil {
		return nil
	}
	
	comp.SubBodies = map[string]*template.Template{}
	extractSubcomponents(comp, body, appParams)
	
	CompCache[name] = comp
	return comp
}

func createTemplate(body, name string, appParams *AppParams, bodyConsts map[string]interface{}) *template.Template {
	templ, err := template.New(name).Funcs(template.FuncMap{
		"evalComp":            EvalComp,
		"generateComp":        GenerateComp,
		"eventHandler":        GenerateEventHandler,
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
	return templ
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

func GenerateEventHandler(comp *CompRuntime, eventTypeCli string, eventTypeSvrOpt ...string) string {
	eventTypeSvr := eventTypeCli
	if len(eventTypeSvrOpt) == 1 {
		eventTypeSvr = eventTypeSvrOpt[0]
	}
	return fmt.Sprintf(" %s=\"se(event,'%s',%d,null)\"", eventTypeCli, eventTypeSvr, comp.Sid())
}

func GenerateEventHandlerWithKey(comp *CompRuntime, eventTypeCli, key string) string {
	return fmt.Sprintf(" %s=\"se(event,'%s',%d,'%s')\"", eventTypeCli, eventTypeCli, comp.Sid(), key)
}

func GenerateComp(parentComp *CompRuntime, sourceChildRef string, genRuntimeRefIf interface{}, context interface{}) string {
	genRuntimeRef := dataxform.IAsString(genRuntimeRefIf)
	genChildRefId := fmt.Sprintf("%s.%s_%s", parentComp.ChildRefId(), sourceChildRef, genRuntimeRef)
	comp := parentComp.GenChildren[genChildRefId]

	if comp == nil {
		logger.Info("genRuntimeRef", genChildRefId)
		srcCompDef := parentComp.Unit.UnitDef.CompsMap[sourceChildRef]
		if srcCompDef == nil {
			logger.Warning("source component not found:", sourceChildRef)
			return ""
		}
		comp = parentComp.Unit.InstantiateComp(srcCompDef, genChildRefId)
		comp.State["parentContext"] = context
		comp.State["parentComp"] = parentComp

		rootCompIf := parentComp.State["rootCompSt"]
		if rootCompIf == nil {
			comp.State["rootCompSt"] = parentComp.State
		} else {
			comp.State["rootCompSt"] = rootCompIf
		}

		parentComp.GenChildren[genChildRefId] = comp
	}

	e := NewEventRuntime(nil, parentComp.Unit, comp, EvtBeforeCompRefresh, "")
	ProcessCompEvent(e)

	sb := &strings.Builder{}
	comp.Render(sb)
	return sb.String()
}
