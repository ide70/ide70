package comp

import (
	"bytes"
	"fmt"
	"github.com/ide70/ide70/dataxform"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"reflect"
)

const UNIT_PATH = "ide70/unit/"

type UnitDef struct {
	RootComp      *CompDef
	CompsMap      map[string]*CompDef
	EventsHandler *UnitDefEventsHandler
	Name          string
}

type UnitDefContext struct {
	idSeq     uint
	appParams *AppParams
	unitDef   *UnitDef
}

func (context *UnitDefContext) getNextId(compType string) string {
	context.idSeq++
	return fmt.Sprintf("%s_%d", compType, context.idSeq)
}

func ParseUnit(name string, appParams *AppParams) *UnitDef {
	contentB, err := ioutil.ReadFile(UNIT_PATH + name + ".yaml")
	if err != nil {
		unitLogger.Error("Unit", name, "not found")
		return nil
	}
	unit := &UnitDef{}
	unit.Name = name
	unit.CompsMap = map[string]*CompDef{}
	unit.EventsHandler = newUnitDefEventsHandler()

	decoder := yaml.NewDecoder(bytes.NewReader(contentB))

	var unitIf interface{}
	err = decoder.Decode(&unitIf)
	if err != nil {
		unitLogger.Error("Unit", name, "failed to decode:", err.Error())
	}

	context := &UnitDefContext{appParams: appParams, unitDef: unit}
	unitIfArr := unitIf.([]interface{})
	for i := 0; i < len(unitIfArr); i++ {
		unitIfArr[i] = dataxform.InterfaceMapToStringMap(unitIfArr[i].(map[interface{}]interface{}))
	}
	for i := 0; i < len(unitIfArr); i++ {
		logger.Info("len:", len(unitIfArr))
		unitIfTag := unitIfArr[i].(map[string]interface{})
		logger.Info("before pcd:", unitIfTag, reflect.TypeOf(unitIfTag))
		compDef := ParseCompDef(unitIfTag, context)
		logger.Info("after pcd")
		if compDef.Props["autoInclude"] != nil {
			unitIfArr = append(unitIfArr, dataxform.IAsArr(compDef.Props["autoInclude"])...)
			logger.Info("adding comps, new len:", len(unitIfArr))
		}
		// handle autoInclude
		unit.CompsMap[compDef.ChildRefId()] = compDef
		if unit.RootComp == nil {
			unit.RootComp = compDef
		}
	}
	logger.Info("components done")

	for _, comp := range unit.CompsMap {
		if comp.Props["children"] == nil {
			continue
		}
		childrenRefs := []string{}
		switch refs := comp.Props["children"].(type) {
		case []interface{}:
			for _, ref := range refs {
				childrenRefs = append(childrenRefs, ref.(string))
			}
		case interface{}:
			childrenRefs = append(childrenRefs, refs.(string))
		}

		logger.Info("children refs:", childrenRefs)
		for _, childRef := range childrenRefs {
			childDef := unit.CompsMap[childRef]
			if childDef == nil {
				// error
			} else {
				comp.Children = append(comp.Children, childDef)
				logger.Info("child add:", childDef)
			}
		}
	}

	return unit
}
