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
}

func init() {
	gob.Register(map[string]interface{}{})
	gob.Register(map[interface{}]interface{}{})
	gob.Register([]interface{}{})
}

func (comp *CompRuntime) Render(writer io.Writer) {
	comp.CompDef.CompType.Body.Execute(writer, comp.State)
}

func InstantiateComp(compDef *CompDef, ctx *UnitCreateContext) *CompRuntime {
	logger.Info("InstantiateComp", compDef)
	comp := &CompRuntime{}
	comp.CompDef = compDef
	logger.Info("RegisterComp", compDef)
	ctx.registerComp(comp)

	// state initially is deep copy of definition properties
	var err error
	comp.State, err = deepCopyMap(compDef.Props)
	comp.State["sid"] = comp.ID
	if err != nil {
		logger.Error(err.Error())
	}
	logger.Info("comp.State", comp.State)

	for _, childDef := range compDef.Children {
		comp.Children = append(comp.Children, InstantiateComp(childDef, ctx))
	}
	comp.State["Children"] = comp.Children

	logger.Info("InstantiateComp-done")

	return comp
}

func (comp *CompRuntime) Sid() int64 {
	return comp.State["sid"].(int64)
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
