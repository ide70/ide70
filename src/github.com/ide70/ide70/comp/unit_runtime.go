package comp

import (
	"github.com/ide70/ide70/app"
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/store"
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

func InstantiateUnit(name string, app *app.Application, appParams *AppParams, passContext *PassContext) *UnitRuntime {
	unitRuntime := &UnitRuntime{}
	unitRuntime.PassContext = passContext
	unitRuntime.Application = app
	unitRuntime.CompRegistry = map[int64]*CompRuntime{}
	unitRuntime.CompByChildRefId = map[string]*CompRuntime{}
	unitDef, has := UnitDefCache[name]

	if !has {
		unitDef = ParseUnit(name, appParams)
		if unitDef == nil {
			return nil
		}
		UnitDefCache[name] = unitDef
		unitLogger.Info("unit definition parsed and cached")
	} else {
		unitLogger.Info("unit definition read from cache")
	}

	unitRuntime.UnitDef = unitDef
	unitRuntime.RootComp = InstantiateComp(unitDef.RootComp, unitRuntime)
	unitRuntime.EventsHandler = newUnitRuntimeEventsHandler(unitRuntime)
	
	unitRuntime.initialCalcs()

	return unitRuntime
}

func (unit *UnitRuntime) initialCalcs() {
	for _,calc := range unit.UnitDef.CalcArr {
		comp := unit.CompByChildRefId[calc.Comp.ChildRefId()]
		e := NewEventRuntime(nil, unit, comp, "calc", "")
		calcResult := unit.EventsHandler.runJs(e, calc.jsCode)
		logger.Info("calc result for", calc.PropertyKey, "is",calcResult,"type:",reflect.TypeOf(calcResult))
		comp.State[calc.PropertyKey]=calcResult
	} 
}

func RefreshUnitDef(name string) {
	delete(UnitDefCache, name)
	logger.Info("refresh unit def:",name)
}

func RefreshCompType(name string) {
	//delete(CompCache, name)
	CompCache = map[string]*CompType{}
	//drop all unit defs
	UnitDefCache = map[string]*UnitDef{}
	logger.Info("refresh comp type:",name)
}

func (unit *UnitRuntime) InstantiateComp(compDef *CompDef, genChildRefId string) *CompRuntime {
	comp := InstantiateComp(compDef, unit)
	comp.State["cr"] = genChildRefId
	// fire initialization event of component

	return comp
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

// process unit lifecycle events
func (unit *UnitRuntime) ProcessEvent(e *EventRuntime) {
	logger.Info("ProcessEvent")
	logger.Info("handlers", unit.UnitDef.EventsHandler.Handlers)
	compDefHandlers := unit.UnitDef.EventsHandler.Handlers[e.TypeCode]
	for _, compDefHandler := range compDefHandlers {
		comp := unit.CompByChildRefId[compDefHandler.CompDef.ChildRefId()]
		logger.Info("On comp", comp.ChildRefId())
		if comp == nil {
			logger.Warning("UnitRuntime ProcessEvent: component not found")
		}
		e.Comp = comp
		compDefHandler.EventHandler.processEvent(e)
	}
}

func (unit *UnitRuntime) CollectStored() map[string]interface{} {
	m := map[string]interface{}{}
	for _, comp := range unit.CompByChildRefId {
		storeKey := dataxform.SIMapGetByKeyAsString(comp.State, "store")
		if storeKey != "" {
			dataxform.SIMapUpdateValue(storeKey, comp.State["value"], m, true)
		}
	}
	log.Info("CollectStored:", m)
	return m
}

func (unit *UnitRuntime) InitializeStored(data map[string]interface{}) {
	for _, comp := range unit.CompByChildRefId {
		storeKey := dataxform.SIMapGetByKeyAsString(comp.State, "store")
		if storeKey != "" {
			comp.State["value"] =
				dataxform.SICollGetNode(storeKey, data)
		}
	}
}

func (unit *UnitRuntime) DBContext() *store.DatabaseContext {
	return unit.Application.Connectors.MainDB
}

func (unit *UnitRuntime) GetParent(sess *Session) *UnitRuntime {
	return sess.UnitCache.ActiveUnits[unit.PassContext.ParentUnitId]
}

