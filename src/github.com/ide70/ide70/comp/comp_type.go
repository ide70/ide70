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
import "reflect"
import "errors"

const COMP_PATH = "ide70/comp/"

var logger = log.Logger{"comp"}
var reSubComp = regexp.MustCompile(`<(\w+)[^<>]+\{\{\.sid\}\}([-\w]+)`)
var nonAccessibleDefinitions = map[string]bool{"unitInterface": true, "body": true}

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
		comp.SubBodies[subCompName] = createTemplate(subCompStr, comp.Name+subCompName, appParams, nil)
	}
}

func cutToClosingTag(s, tagName string) string {
	r := regexp.MustCompile(fmt.Sprintf(`<%s|<\/%s>`, tagName, tagName))
	subs := r.FindAllStringSubmatch(s, -1)
	subIdxs := r.FindAllStringSubmatchIndex(s, -1)
	openTags := 0
	for i, sub := range subs {
		if strings.HasPrefix(sub[0], "</") {
			openTags--
		} else {
			openTags++
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

	for k, v := range module.Def {
		if !nonAccessibleDefinitions[k] {
			comp.AccessibleDef[k] = v
		}
	}

	parseDefaultValues(dataxform.SIMapGetByKeyChainAsMap(module.Def, "unitInterface.properties"), comp.AccessibleDef)
	parseDefaultValues(dataxform.SIMapGetByKeyAsMap(module.Def, "privateProperties"), comp.AccessibleDef)

	// TODO: list of non-accessible definitions
	/*comp.AccessibleDef["eventHandlers"] = module.Def["eventHandlers"]
	comp.AccessibleDef["autoInclude"] = module.Def["autoInclude"]
	comp.AccessibleDef["css"] = module.Def["css"]
	comp.AccessibleDef["injectRootComp"] = module.Def["injectRootComp"]
	comp.AccessibleDef["injectToComp"] = module.Def["injectToComp"]*/

	comp.Body = createTemplate(body, name, appParams, bodyConsts)
	if comp.Body == nil {
		return nil
	}

	comp.SubBodies = map[string]*template.Template{}
	extractSubcomponents(comp, body, appParams)

	CompCache[name] = comp
	return comp
}

func parseDefaultValues(def map[string]interface{}, dest map[string]interface{}) {
	dataxform.IApplyFnToNodes(def, func(entry dataxform.CollectionEntry) {
		if entry.Key() == "default" {
			parentKey := entry.Parent().LinearKey()
			dataxform.SIMapUpdateValue(parentKey, entry.Value(), dest, false)
			logger.Warning("default value", parentKey, entry.Value())
		}
	})
}

func createTemplate(body, name string, appParams *AppParams, bodyConsts map[string]interface{}) *template.Template {
	templ, err := template.New(name).Funcs(template.FuncMap{
		"passRoot":            passRoot,
		"htmlId":              htmlId,
		"printVar":            printVar,
		"dict":                dict,
		"evalComp":            EvalComp,
		"generateComp":        GenerateComp,
		"eventHandler":        GenerateEventHandler,
		"eventHandlerJs":      GenerateEventHandlerJs,
		"eventHandlerWithKey": GenerateEventHandlerWithKey,
		"numRange":            numRange,
		"numRangeOpenEnd":     numRangeOpenEnd,
		"linearContext":       LinearContext,
		"generateSubComp":     GenerateSubComp,
		"dropSubComp":         DropSubComp,
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

func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func passRoot(current, root interface{}) map[string]interface{} {
	return map[string]interface{}{"c": current, "r": root}
}

func numRange(startI, endI interface{}) (stream chan int) {
	stream = make(chan int)
	start := dataxform.IAsInt(startI)
	end := dataxform.IAsInt(endI)

	go func() {
		for i := start; i <= end; i++ {
			stream <- i
		}
		close(stream)
	}()
	return
}

func numRangeOpenEnd(startI, endI interface{}) (stream chan int) {
	stream = make(chan int)
	start := dataxform.IAsInt(startI)
	end := dataxform.IAsInt(endI)

	go func() {
		for i := start; i < end; i++ {
			stream <- i
		}
		close(stream)
	}()
	return
}

func htmlId(sI interface{}) string {
	logger.Info("htmlId")
	s := dataxform.IAsString(sI)
	logger.Info("htmlId", s)
	s = strings.ReplaceAll(s, "/", "_")
	return s
}

func printVar(i interface{}) string {
	fmt.Println("i:", reflect.TypeOf(i))
	fmt.Println("iv:", i)
	return ""
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
	jsObjectToPass := "null"
	if len(eventTypeSvrOpt) >= 1 {
		eventTypeSvr = eventTypeSvrOpt[0]
	}
	if len(eventTypeSvrOpt) >= 2 {
		jsObjectToPass = eventTypeSvrOpt[1]
	}
	return fmt.Sprintf(" %s=\"se(event,'%s',%d,%s)\"", eventTypeCli, eventTypeSvr, comp.Sid(), jsObjectToPass)
}

func GenerateEventHandlerJs(comp *CompRuntime, eventType, valueJs string) string {
	return fmt.Sprintf(" %s=\"se(event,'%s',%d,%s)\"", eventType, eventType, comp.Sid(), valueJs)
}

func GenerateEventHandlerWithKey(comp *CompRuntime, eventTypeCli, eventTypeSvr, keyIf interface{}) string {
	key := dataxform.IAsString(keyIf)
	/*logger.Info("GenerateEventHandlerWithKey")
	logger.Info(comp)
	logger.Info(eventTypeCli)
	logger.Info(eventTypeSvr)
	logger.Info(key)*/
	return fmt.Sprintf(" %s=\"se(event,'%s',%d,'%s')\"", eventTypeCli, eventTypeSvr, comp.Sid(), key)
}

type GenerationContext struct {
	index                     int
	key                       string
	parentComp                *CompRuntime
	childRef                  string
	generateChildRef          func(gc *GenerationContext, childRef string) string
	generateChildRefWithIndex func(gc *GenerationContext, childRef string, index interface{}) string
	generateChildRefPrefix    func(gc *GenerationContext) string
	generateStoreKey          func(gc *GenerationContext, child *CompRuntime) string
}

func LinearContext(parentComp *CompRuntime, childRefIf interface{}, indexIf interface{}) *GenerationContext {
	logger.Info("LinearContext")
	childRef := ""
	switch childRefT := childRefIf.(type) {
	case *CompRuntime:
		childRef = childRefT.ChildRefId()
	default:
		childRef = dataxform.IAsString(childRefIf)
	}
	index := dataxform.IAsInt(indexIf)
	gc := &GenerationContext{index: index, parentComp: parentComp, childRef: childRef, generateChildRef: generateChildRefLinear, generateStoreKey: generateStoreKeyLinear, generateChildRefPrefix: generateChildRefPrefixLinear, generateChildRefWithIndex: generateChildRefLinearWithIndex}
	logger.Info("GenerationContext:", gc)
	return gc
}

func generateChildRefLinear(gc *GenerationContext, childRef string) string {
	return fmt.Sprintf("%s_%d.%s", gc.parentComp.ChildRefId(), gc.index, childRef)
}

func generateChildRefLinearWithIndex(gc *GenerationContext, childRef string, indexIf interface{}) string {
	index := dataxform.IAsInt(indexIf)
	return fmt.Sprintf("%s_%d.%s", gc.parentComp.ChildRefId(), index, childRef)
}

func generateChildRefPrefixLinear(gc *GenerationContext) string {
	return fmt.Sprintf("%s_%d.", gc.parentComp.ChildRefId(), gc.index)
}

func generateStoreKeyLinear(gc *GenerationContext, child *CompRuntime) string {
	if gc.parentComp.State["store"] == nil {
		return fmt.Sprintf("%s[%d]", child.State["store"], gc.index)
	}
	return fmt.Sprintf("%s[%d].%s", gc.parentComp.State["store"], gc.index, child.State["store"])
}

func GenerateSubComp(gc *GenerationContext) string {
	logger.Info("GenerateSubComp:", gc)
	genChildRefId := gc.generateChildRef(gc, gc.childRef)
	gc.parentComp.State["keepExistingGenChildren"] = true

	comp := gc.parentComp.GenChildren[genChildRefId]

	logger.Info("comp is new:", (comp == nil))

	if comp == nil {
		logger.Info("genRuntimeRef", genChildRefId)
		srcCompDef := gc.parentComp.Unit.UnitDef.CompsMap[childRefLastTag(gc.childRef)]
		if srcCompDef == nil {
			logger.Warning("source component not found:", gc.childRef)
			return ""
		}
		comp = gc.parentComp.Unit.InstantiateGeneratedComp(srcCompDef, gc)

		rootCompIf := gc.parentComp.State["rootCompSt"]
		if rootCompIf == nil {
			comp.State["rootCompSt"] = gc.parentComp.State
		} else {
			comp.State["rootCompSt"] = rootCompIf
		}

		gc.parentComp.GenChildren[genChildRefId] = comp

	}

	e := NewEventRuntime(nil, gc.parentComp.Unit, comp, EvtBeforeCompRefresh, "")
	ProcessCompEvent(e)

	logger.Info("CRs:", gc.parentComp.Unit.CompByChildRefId)

	sb := &strings.Builder{}
	comp.Render(sb)
	return sb.String()
}

func DropSubComp(gc *GenerationContext) string {
	logger.Info("DropSubComp:", gc)
	genChildRefId := gc.generateChildRef(gc, gc.childRef)

	comp := gc.parentComp.GenChildren[genChildRefId]

	logger.Info("comp exist:", (comp != nil))

	if comp != nil {
		delete(gc.parentComp.GenChildren, genChildRefId)
		comp.Drop()
	}

	return ""
}

func childRefLastTag(childRef string) string {
	tags := strings.Split(childRef, ".")
	return tags[len(tags)-1]
}

func GenerateComp(parentComp *CompRuntime, sourceChildRef string, genRuntimeRefIf interface{}, context interface{}) string {
	logger.Info("GenerateComp", sourceChildRef)
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
