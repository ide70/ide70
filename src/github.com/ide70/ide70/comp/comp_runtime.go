package comp

import (
	"bytes"
	"encoding/gob"
	"io"
)

// a component instance
type CompRuntime struct {
	CompDef  *CompDef
	ID       int64
	State    map[string]interface{}
	Children []*CompRuntime
	// on-the-fly generated sub-components
	GenChilden map[string]*CompRuntime
	Unit     *UnitRuntime
}

func init() {
	gob.Register(map[string]interface{}{})
	gob.Register(map[interface{}]interface{}{})
	gob.Register([]interface{}{})
}

func (comp *CompRuntime) Render(writer io.Writer) {
	//buf := &bytes.Buffer{}
	//comp.CompDef.CompType.Body.Execute(buf, comp.State)
	//logger.Info(buf.String())
	if len(comp.GenChilden) > 0 {
		comp.GenChilden = map[string]*CompRuntime{}
	}
	comp.CompDef.CompType.Body.Execute(writer, comp.State)
}

func InstantiateComp(compDef *CompDef, unit *UnitRuntime) *CompRuntime {
	logger.Info("InstantiateComp", compDef.ChildRefId(), compDef.CompType.Name)
	comp := &CompRuntime{}
	comp.CompDef = compDef
	comp.Unit = unit
	var err error
	comp.State, err = deepCopyMap(compDef.Props)
	if err != nil {
		logger.Error(err.Error())
	}
	logger.Info("RegisterComp", compDef)
	unit.registerComp(comp)

	comp.GenChilden = map[string]*CompRuntime{}
	// state initially is deep copy of definition properties
	comp.State["sid"] = comp.ID
	logger.Info("comp.State", comp.State)

	for _, childDef := range compDef.Children {
		comp.Children = append(comp.Children, InstantiateComp(childDef, unit))
	}
	comp.State["Children"] = comp.Children
	comp.State["This"] = comp

	logger.Info("InstantiateComp-done")

	return comp
}

func (comp *CompRuntime) Sid() int64 {
	return comp.State["sid"].(int64)
}

func (comp *CompRuntime) ChildRefId() string {
	return comp.State["cr"].(string)
}

func deepCopyMap(m map[string]interface{}) (map[string]interface{}, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	err := enc.Encode(m)
	if err != nil {
		return nil, err
	}
	var copy map[string]interface{}
	err = dec.Decode(&copy)
	if err != nil {
		return nil, err
	}
	return copy, nil
}
