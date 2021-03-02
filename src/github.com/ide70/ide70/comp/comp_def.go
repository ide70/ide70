package comp

import (
	"github.com/ide70/ide70/dataxform"
)

// a component instance
type CompDef struct {
	CompType      *CompType
	Children      []*CompDef
	ChildRefId    string
	Props         map[string]interface{}
	EventsHandler *CompDefEventsHandler
}

func ParseCompDef(def map[string]interface{}, context *UnitDefContext) *CompDef {
	logger.Info("ParseCompDef", def)
	compDef := &CompDef{}
	compTypeName := def["compType"].(string)
	compDef.CompType = GetCompType(compTypeName, context.appParams)
	compDef.Props = def
	
	// TODO: lista merge nem az igazi
	dataxform.SIMapInjectDefaults(compDef.CompType.AccessibleDef, compDef.Props)

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

	compDef.EventsHandler = ParseEventHandlers(def, compDef.CompType.EventsHandler)

	logger.Info("ParseCompDef end")
	return compDef
}

func ParseEventHandlers(def map[string]interface{}, superEventsHandler *CompDefEventsHandler) *CompDefEventsHandler {
	eventsHandler := newEventsHandler()

	logger.Info("ParseEventHandlers super:", superEventsHandler)
	if superEventsHandler != nil {
		for eventType, handler := range superEventsHandler.Handlers {
			eventsHandler.AddHandler(eventType, handler)
		}
	}

	logger.Info("ParseEventHandlers def:", def)
	for eventType, eventPropsIf := range dataxform.SIMapGetByKeyAsMap(def, "eventHandlers") {
		eventProps := dataxform.AsSIMap(eventPropsIf)
		eventAction := dataxform.SIMapGetByKeyAsString(eventProps, "action")
		eventHandler := newEventHandler()
		eventHandler.JsCode = eventAction
		eventsHandler.AddHandler(eventType, eventHandler)
	}
	logger.Info("eventsHandler created:", eventsHandler)
	return eventsHandler
}
