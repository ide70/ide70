package comp

import (
	"github.com/ide70/ide70/dataxform"
)

// a component instance
type CompDef struct {
	CompType *CompType
	Children []*CompDef
	ChildRefId string
	Props    map[string]interface{}
	EventsHandler *CompDefEventsHandler
}

func ParseCompDef(def map[string]interface{}, context *UnitDefContext) *CompDef {
	logger.Info("ParseCompDef", def)
	compDef := &CompDef{}
	compTypeName := def["compType"].(string)
	compDef.CompType = GetCompType(compTypeName, context.appParams)
	compDef.Props = def
	logger.Info("ParseCompDef id before")
	id := ""
	if def["cr"] != nil {
		id = def["cr"].(string)
	}
	logger.Info("ParseCompDef id", id)
	if id == "" {
		id = context.getNextId(compTypeName)
	}
	logger.Info("ParseCompDef id gen", id)
	compDef.ChildRefId = id
	
	compDef.EventsHandler = newEventsHandler()
	for _,eventHandlerIf := range dataxform.SIMapGetByKeyAsList(def, "eventHandlers") {
		eventhandlerProps := dataxform.AsSIMap(eventHandlerIf)
		eventType := dataxform.SIMapGetByKeyAsString(eventhandlerProps, "event")
		eventAction := dataxform.SIMapGetByKeyAsString(eventhandlerProps, "action")
		eventHandler := newEventHandler()
		eventHandler.JsCode = eventAction
		compDef.EventsHandler.AddHandler(eventType, eventHandler)
	}
	
	logger.Info("ParseCompDef end")
	return compDef
}
