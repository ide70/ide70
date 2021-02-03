package comp

import (
	"github.com/ide70/ide70/util/log"
	"io"
)

var unitLogger = log.Logger{"unit"}

type UnitRuntime struct {
	UnitDef  *UnitDef
	RootComp *CompRuntime
	CompRegistry map[int64] *CompRuntime
	CompByChildRefId map[string] *CompRuntime
	EventsHandler *UnitRuntimeEventsHandler
}

type AppParams struct {
	PathStatic string
	Path string
	RuntimeID string
}

type UnitCreateContext struct {
	IDSeq int64
	UnitRuntime *UnitRuntime
}

func (ctx *UnitCreateContext) nextId() int64 {
	ctx.IDSeq++
	return ctx.IDSeq
}

func (ctx *UnitCreateContext) registerComp(compRuntime *CompRuntime) {
	compRuntime.ID = ctx.nextId()
	ctx.UnitRuntime.CompRegistry[compRuntime.ID]=compRuntime;
	ctx.UnitRuntime.CompByChildRefId[compRuntime.CompDef.ChildRefId]=compRuntime;
}

func InstantiateUnit(name string, appParams *AppParams) *UnitRuntime {
	unitRuntime := &UnitRuntime{}
	unitRuntime.CompRegistry = map[int64] *CompRuntime{}
	unitRuntime.CompByChildRefId = map[string] *CompRuntime{}
	unitDef, has := UnitCache[name]

	if !has {
		unitDef = ParseUnit(name, appParams)
		if unitDef == nil {
			return nil
		}
		UnitCache[name] = unitDef
		unitLogger.Info("unit definition parsed and cached")
	} else {
		unitLogger.Info("unit definition read from cache")
	}

	unitRuntime.UnitDef = unitDef
	ctx := &UnitCreateContext{UnitRuntime: unitRuntime}
	unitRuntime.RootComp = InstantiateComp(unitDef.RootComp, ctx)
	unitRuntime.EventsHandler = newUnitRuntimeEventsHandler(unitRuntime)

	return unitRuntime
}

func (unit *UnitRuntime) Render(writer io.Writer) {
	unit.RootComp.Render(writer)
}

func (unit *UnitRuntime) AssignID(id string) {
	unit.RootComp.State["_unitID"] = id
}
