package comp

import "github.com/ide70/ide70/util/log"
import "io/ioutil"
import "text/template"
import "strings"

const COMP_PATH = "ide70/comp/"

var logger = log.Logger{"comp"}

// An user defined component
type CompType struct {
	Name string
	Body *template.Template
}

func parseCompType(name string, appParams *AppParams) *CompType {
	logger.Info("parseCompType", name)
	contentB, err := ioutil.ReadFile(COMP_PATH + name + ".json")
	if err != nil {
		logger.Error("Component", name, "not found")
		return nil
	}
	content := string(contentB)
	comp := &CompType{}
	comp.Name = name
	comp.Body, err = template.New(name).Funcs(template.FuncMap{
		"evalComp": EvalComp,
		"app": func() *AppParams {
			return appParams
		},
	}).Parse(content)
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
