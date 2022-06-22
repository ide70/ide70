package comp

import (
	"bytes"
	"encoding/gob"
	"io"
	"github.com/ide70/ide70/dataxform"
)

// a component instance
type CompRuntime struct {
	CompDef  *CompDef
	ID       int64
	State    map[string]interface{}
	Children []*CompRuntime
	// on-the-fly generated sub-components
	GenChildren map[string]*CompRuntime
	Unit        *UnitRuntime
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
	if len(comp.GenChildren) > 0 && !dataxform.SIMapGetByKeyAsBoolean(comp.State, "keepExistingGenChildren") {
		comp.GenChildren = map[string]*CompRuntime{}
	}
	comp.CompDef.CompType.Body.Execute(writer, comp.State)
}

func (comp *CompRuntime) RenderSub(subCompName string, writer io.Writer) {
	subTemplate := comp.CompDef.CompType.SubBodies[subCompName]
	logger.Info("RenderSub")
	if subTemplate != nil {
		logger.Info("RenderSub has template")
		if comp.IsEventDefined(EvtBeforeCompRefresh) {
			logger.Info("event defined")
			e := NewEventRuntime(nil, comp.Unit, comp, EvtBeforeCompRefresh, "")
			ProcessCompEvent(e)
		} else {
			logger.Info("event not defined")
		}

		subTemplate.Execute(writer, comp.State)
	}
}

func InstantiateComp(compDef *CompDef, unit *UnitRuntime, gc *GenerationContext) *CompRuntime {
	logger.Info("InstantiateComp", compDef.ChildRefId(), compDef.CompType.Name)
	comp := &CompRuntime{}
	comp.CompDef = compDef
	comp.Unit = unit
	var err error
	comp.State, err = deepCopyMap(compDef.Props)
	if err != nil {
		logger.Error(err.Error())
	}
	logger.Debug("RegisterComp", compDef)

	if gc != nil {
		// override fixed cr and store by generation context
		comp.State["cr"] = gc.generateChildRef(gc, comp.ChildRefId())
		comp.State["crPrefix"] = gc.generateChildRefPrefix(gc)
		if _, has := comp.State["store"]; has {
			comp.State["store"] = gc.generateStoreKey(gc, comp)
		}
	}

	unit.registerComp(comp)

	comp.GenChildren = map[string]*CompRuntime{}
	// state initially is deep copy of definition properties
	comp.State["sid"] = comp.ID
	logger.Debug("comp.State", comp.State)

	for _, childDef := range compDef.Children {
		comp.Children = append(comp.Children, InstantiateComp(childDef, unit, gc))
	}
	comp.State["Children"] = comp.Children
	comp.State["This"] = comp
	
	/*if gc != nil {
		unit.ProcessInitEventsComp(comp)
	}*/

	logger.Info("InstantiateComp-done")

	return comp
}

func (comp *CompRuntime) Sid() int64 {
	return comp.State["sid"].(int64)
}

func (comp *CompRuntime) ChildRefId() string {
	return comp.State["cr"].(string)
}

func (comp *CompRuntime) IsEventDefined(eventType string) bool {
	return comp.CompDef.EventsHandler.Handlers[eventType] != nil
}

func (comp *CompRuntime) Drop() {
	delete(comp.Unit.CompByChildRefId,comp.ChildRefId())
	delete(comp.Unit.CompRegistry,comp.ID)
	for _,child := range comp.Children {
		child.Drop()
	}
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
