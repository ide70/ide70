package comp

import (
	"bytes"
	"fmt"
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/dataxform"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"reflect"
)

const UNIT_PATH = "ide70/unit/"

type Calc struct {
	Comp        *CompDef
	PropertyKey string
	jsCode      string
}

type UnitDef struct {
	RootComp        *CompDef
	CompsMap        map[string]*CompDef
	UnattachedComps []*CompDef
	EventsHandler   *UnitDefEventsHandler
	Name            string
	CalcArr         []*Calc
	Props         map[string]interface{}
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
	unit.CalcArr = []*Calc{}
	unit.Props = map[string]interface{}{}

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
	
	compsOrder := []*CompDef{}
	
	for i := 0; i < len(unitIfArr); i++ {
		logger.Debug("len:", len(unitIfArr))
		unitIfTag := unitIfArr[i].(map[string]interface{})
		logger.Debug("before pcd:", unitIfTag, reflect.TypeOf(unitIfTag))
		compDef := ParseCompDef(unitIfTag, context)
		logger.Debug("after pcd")
		if compDef.Props["autoInclude"] != nil {
			unitIfArr = append(unitIfArr, api.IAsArr(compDef.Props["autoInclude"])...)
			logger.Debug("adding comps, new len:", len(unitIfArr))
		}
		// handle autoInclude
		unit.CompsMap[compDef.ChildRefId()] = compDef
		compsOrder = append(compsOrder, compDef)
		
		if unit.RootComp == nil {
			unit.RootComp = compDef
		}
	}
	logger.Debug("components done")

	for _, comp := range unit.CompsMap {
		if comp.Props["injectRootComp"] == nil {
			continue
		}
		defs := api.AsSIMap(comp.Props["injectRootComp"])
		logger.Debug("injectRootComp defs:", defs)
		api.SIMapInjectDefaults(defs, unit.RootComp.Props)
	}

	for _, comp := range unit.CompsMap {
		logger.Debug("itc check")
		if comp.Props["injectToComp"] == nil {
			continue
		}
		logger.Debug("injectToComp")
		injectDefsArr := api.IAsArr(comp.Props["injectToComp"])
		logger.Debug("injectDefsArr", injectDefsArr)
		for _, injectDefIf := range injectDefsArr {
			injectDef := api.AsSIMap(injectDefIf)
			logger.Debug("injectDef", injectDef)

			filter := api.IAsSIMap(injectDef["filter"])

			targetCompList := evalFilter(filter, unit.CompsMap)

			if len(targetCompList) == 0 {
				cr := api.IAsString(injectDef["cr"])
				logger.Debug("cr", cr)
				targetComp := unit.CompsMap[cr]
				logger.Debug("targetComp", targetComp.ChildRefId())
				if targetComp == nil {
					continue
				}
				targetCompList = append(targetCompList, targetComp)
			}

			defs := api.IAsSIMap(injectDef["defs"])
			logger.Debug("defs", defs)
			toCopyArr := api.IAsArr(injectDef["copy"])
			logger.Debug("toCopyArr", toCopyArr)

			for _, targetComp := range targetCompList {
				if len(defs) > 0 {
					logger.Debug("adding defs: ", targetComp.ChildRefId(), defs)
					logger.Debug("before: ", targetComp.Props["eventHandlers"])
					api.SIMapInjectDefaults(defs, targetComp.Props)
					logger.Debug("after: ", targetComp.Props["eventHandlers"])
				}

				if toCopyArr != nil {
					for _, toCopyIf := range toCopyArr {
						toCopy := api.IAsString(toCopyIf)
						targetComp.Props[toCopy] = comp.Props[toCopy]
						logger.Debug("copying: ", toCopy, comp.Props[toCopy])
					}
					api.SIMapInjectDefaults(defs, targetComp.Props)
				}
			}
		}

	}

	attachedCompMap := map[string]*CompDef{}
	attachedCompMap[unit.RootComp.ChildRefId()] = unit.RootComp

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

		logger.Debug("children refs:", childrenRefs)
		for _, childRef := range childrenRefs {
			childDef := unit.CompsMap[childRef]
			if childDef == nil {
				// error
			} else {
				comp.Children = append(comp.Children, childDef)
				logger.Debug("child add:", childDef)
				attachedCompMap[childRef] = childDef
			}
		}
	}
	
	// parse event handler only when component property set is complete
	for _, comp := range compsOrder {
		comp.ParseEventHandlers(context)
	}

	childrenIf := unit.RootComp.Props["tree"]
	logger.Debug("tree scruct:", childrenIf)
	if childrenIf != nil {
		processCompTree(unit, unit.RootComp, api.IAsArr(childrenIf), attachedCompMap)
	}

	for cr, comp := range unit.CompsMap {
		if attachedCompMap[cr] == nil {
			unit.UnattachedComps = append(unit.UnattachedComps, comp)
			logger.Debug("unattached:", cr)
		}
	}

	return unit
}

func evalFilter(filter map[string]interface{}, comps map[string]*CompDef) []*CompDef {
	resComps := []*CompDef{}
	propFilter := api.SIMapGetByKeyAsString(filter, "hasProp")
	for _, comp := range comps {
		if !api.IsEmpty(comp.Props[propFilter]) {
			resComps = append(resComps, comp)
		}
	}
	return resComps
}

func registerChild(unit *UnitDef, comp *CompDef, childRef string, attachedCompMap map[string]*CompDef) *CompDef {
	childDef := unit.CompsMap[childRef]
	if childDef == nil {
		logger.Error("tree: childRef not found:", childRef)
		return nil
	} else {
		comp.Children = append(comp.Children, childDef)
		attachedCompMap[childRef] = childDef
		return childDef
	}
}

func processCompTree(unit *UnitDef, comp *CompDef, children []interface{}, attachedCompMap map[string]*CompDef) {
	for _, child := range children {
		switch Tchild := child.(type) {
		case string:
			registerChild(unit, comp, Tchild, attachedCompMap)
		case map[string]interface{}:
			for childRef, subNode := range Tchild {
				childComp := registerChild(unit, comp, childRef, attachedCompMap)
				if childComp != nil {
					processCompTree(unit, childComp, api.IAsArr(subNode), attachedCompMap)
				}
			}
		}
	}
}

func (unit *UnitDef) getInitialEventCodes() []string{
	eventCodeList := api.IArrToStringArr(api.SIMapGetByKeyAsList(unit.Props, "initialEventList"))
	if len(eventCodeList) == 0 {
		eventCodeList = append(eventCodeList, EvtUnitCreate)
	}
	return eventCodeList
}

func (unit *UnitDef) getPostRenderEventCodes() []string{
	eventCodeList := api.IArrToStringArr(api.SIMapGetByKeyAsList(unit.Props, "postRenderEventList"))
	return eventCodeList
}
