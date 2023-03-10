package comp

import (
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/app"
	"github.com/ide70/ide70/util/log"
	"io"
	"reflect"
)

var unitLogger = log.Logger{"unit"}

type UnitRuntime struct {
	PassContext      *PassContext
	UnitDef          *UnitDef
	RootComp         *CompRuntime
	CompRegistry     map[int64]*CompRuntime
	CompByChildRefId map[string]*CompRuntime
	EventsHandler    *UnitRuntimeEventsHandler
	Application      *app.Application
	IDSeq            int64
	appParams *AppParams
	session			*Session
}

type AppParams struct {
	PathStatic string
	Path       string
	RuntimeID  string
}

func (unit *UnitRuntime) nextId() int64 {
	unit.IDSeq++
	return unit.IDSeq
}

func (unit *UnitRuntime) registerComp(compRuntime *CompRuntime) {
	compRuntime.ID = unit.nextId()
	unit.CompRegistry[compRuntime.ID] = compRuntime
	unit.CompByChildRefId[compRuntime.ChildRefId()] = compRuntime
}

func InstantiateUnit(name string, app *app.Application, appParams *AppParams, passContext *PassContext, session *Session) *UnitRuntime {
	unitRuntime := &UnitRuntime{}
	unitRuntime.appParams = appParams
	unitRuntime.PassContext = passContext
	unitRuntime.Application = app
	unitRuntime.CompRegistry = map[int64]*CompRuntime{}
	unitRuntime.CompByChildRefId = map[string]*CompRuntime{}
	unitRuntime.session = session
	unitDef, has := UnitDefCache[name]

	if !has {
		unitDef = ParseUnit(name, appParams)
		if unitDef == nil {
			return nil
		}
		UnitDefCache[name] = unitDef
		unitLogger.Debug("unit definition parsed and cached")
	} else {
		unitLogger.Debug("unit definition read from cache")
	}

	unitRuntime.UnitDef = unitDef
	unitRuntime.RootComp = InstantiateComp(unitDef.RootComp, unitRuntime, nil)
	unitRuntime.RootComp.State["_accessPrefix"] = session.GetAccessPrefix()
	for _, comp := range unitDef.UnattachedComps {
		InstantiateComp(comp, unitRuntime, nil)
	}
	unitRuntime.EventsHandler = newUnitRuntimeEventsHandler(unitRuntime)

	unitRuntime.initialCalcs()

	return unitRuntime
}

func (unit *UnitRuntime) initialCalcs() {
	for _, calc := range unit.UnitDef.CalcArr {
		comp := unit.CompByChildRefId[calc.Comp.ChildRefId()]
		e := NewEventRuntime(nil, unit, comp, "calc", "")
		calcResult := unit.EventsHandler.runJs(e, calc.jsCode)
		logger.Debug("calc result for", calc.PropertyKey, "is", calcResult, "type:", reflect.TypeOf(calcResult))
		comp.State[calc.PropertyKey] = calcResult
	}
}

func RefreshUnitDef(name string) {
	delete(UnitDefCache, name)
	logger.Debug("refresh unit def:", name)
}

func RefreshCompType(name string) {
	//delete(CompCache, name)
	CompCache = map[string]*CompType{}
	//drop all unit defs
	UnitDefCache = map[string]*UnitDef{}
	logger.Debug("refresh comp type:", name)
}

func (unit *UnitRuntime) InstantiateComp(compDef *CompDef, genChildRefId string) *CompRuntime {
	comp := InstantiateComp(compDef, unit, nil)
	comp.State["cr"] = genChildRefId
	// fire initialization event of component

	return comp
}

func (unit *UnitRuntime) InstantiateGeneratedComp(compDef *CompDef, gc *GenerationContext) *CompRuntime {
	comp := InstantiateComp(compDef, unit, gc)
	// fire initialization event of component tree
	unit.processInitEventsCompTree(comp)

	return comp
}

func (unit *UnitRuntime) processInitEventsCompTree(comp *CompRuntime) {
	unit.ProcessInitEventsComp(comp)
	for _,subComp := range comp.Children {
		unit.processInitEventsCompTree(subComp)
	}
}



func (unit *UnitRuntime) Render(writer io.Writer) {
	unit.RootComp.Render(writer)
}

func (unit *UnitRuntime) AssignID(id string) {
	unit.RootComp.State["_unitID"] = id
}

func (unit *UnitRuntime) getID() string {
	return unit.RootComp.State["_unitID"].(string)
}

func (unit *UnitRuntime) ProcessInitEventsComp(comp *CompRuntime) {
	eventCodeList := unit.UnitDef.getInitialEventCodes()
	//logger.Debug("IEC eventCodeList for",comp.ChildRefId(), eventCodeList)
	for _, eventCode := range eventCodeList {
		e := NewEventRuntime(nil, unit, comp, eventCode, "")
		ProcessCompEvent(e)
	}
}

func (unit *UnitRuntime) ProcessInitEvents(sess *Session) {
	eventCodeList := unit.UnitDef.getInitialEventCodes()
	//logger.Debug("eventCodeList:",eventCodeList)
	for _, eventCode := range eventCodeList {
		e := NewEventRuntime(sess, unit, nil, eventCode, "")
		unit.ProcessEvent(e)
	}
}

func (unit *UnitRuntime) ProcessPostRenderEvents(sess *Session) {
	eventCodeList := unit.UnitDef.getPostRenderEventCodes()
	logger.Debug("post render eventCodeList:",eventCodeList)
	for _, eventCode := range eventCodeList {
		e := NewEventRuntime(sess, unit, nil, eventCode, "")
		unit.ProcessEvent(e)
	}
}

// process unit lifecycle events
func (unit *UnitRuntime) ProcessEvent(e *EventRuntime) {
	logger.Debug("ProcessEvent:", e.TypeCode)
	logger.Debug("handlers", unit.UnitDef.EventsHandler.Handlers)
	compDefHandlers := unit.UnitDef.EventsHandler.Handlers[e.TypeCode]
	for _, compDefHandler := range compDefHandlers {
		comp := unit.CompByChildRefId[compDefHandler.CompDef.ChildRefId()]
		//logger.Debug("On comp", comp.ChildRefId())
		if comp == nil {
			logger.Warning("UnitRuntime ProcessEvent: component not found")
		}
		e.Comp = comp
		eh := compDefHandler.EventHandler
		for {
			eh.processEvent(e)
			if len(e.ResponseAction.forward) > 0 {
				logger.Debug("unit event forward to type:" + e.ResponseAction.forward[0].EventType)
				e.Comp = e.ResponseAction.forward[0].To
				e.TypeCode = e.ResponseAction.forward[0].EventType
				e.Params = e.ResponseAction.forward[0].Params
				logger.Debug("with params:", e.Params)
				e.ResponseAction.forward = e.ResponseAction.forward[1:]
				eh = e.Comp.CompDef.EventsHandler.Handlers[e.TypeCode]
				continue
			}
			break
		}
	}
}

func (unit *UnitRuntime) CollectStored() map[string]interface{} {
	m := map[string]interface{}{}
	for _, comp := range unit.CompByChildRefId {
		storeKey := api.SIMapGetByKeyAsString(comp.State, "store")
		// subfieldStore handling
		if storeKey != "" {
			logger.Debug("comp:", comp.ChildRefId(), " key:", storeKey, " value:", comp.State["value"])
			if value, has := comp.State["value"]; has {
				api.SIMapUpdateValue(storeKey, value, m, true)
			}
		}
	}
	log.Debug("CollectStored:", m)
	return m
}

func (unit *UnitRuntime) InitializeStored(data map[string]interface{}) {
	for _, comp := range unit.CompByChildRefId {
		storeKey := api.SIMapGetByKeyAsString(comp.State, "store")
		if storeKey != "" {
			comp.State["value"] =
				api.SICollGetNode(storeKey, data)
			log.Debug("datamap vt:", reflect.TypeOf(comp.State["value"]), storeKey)
		}
	}
}

/*func (unit *UnitRuntime) InitializeStoredComp(comp *CompRuntime, data map[string]interface{}) {

	storeKey := api.SIMapGetByKeyAsString(comp.State, "store")
	if storeKey == "" {
		return
	}
	comp.State["value"] =
		api.SICollGetNode(storeKey, data)
	log.Debug("datamap vt:", reflect.TypeOf(comp.State["value"]), storeKey)

}*/

func (unit *UnitRuntime) DBContext() *api.DatabaseContext {
	return unit.Application.Connectors.MainDB
}

func (unit *UnitRuntime) GetParent(sess *Session) *UnitRuntime {
	if unit.PassContext == nil {
		return nil
	}
	return sess.UnitCache.ActiveUnits[unit.PassContext.ParentUnitId]
}

