package comp

import (
	"github.com/ide70/ide70/dataxform"
	"strings"
)

// a component instance
type CompDef struct {
	CompType      *CompType
	Children      []*CompDef
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
	if def["cr"] == nil {
		def["cr"] = context.getNextId(compTypeName)
	}
	logger.Info("ParseCompDef id", def["cr"])

	compDef.EventsHandler = ParseEventHandlers(def, compDef.CompType.EventsHandler, context, compDef)
	
	parseCalcs(def, context, compDef)

	logger.Info("ParseCompDef end")
	return compDef
}

func parseCalcs(def map[string]interface{}, context *UnitDefContext, compDef *CompDef) {
	dataxform.IApplyFnToNodes(def, func(entry dataxform.CollectionEntry) {
			if entry.Key() == "calc" {
				value := dataxform.IAsString(entry.Value())
				if value != "" {
					calc := &Calc{Comp: compDef, PropertyKey: entry.Parent().Key(), jsCode: value}
					logger.Info("calc added:",*calc)
					context.unitDef.CalcArr = append(context.unitDef.CalcArr, calc)
				}
			}
	})
}

func ParseEventHandlers(def map[string]interface{}, superEventsHandler *CompDefEventsHandler, context *UnitDefContext, compDef *CompDef) *CompDefEventsHandler {
	eventsHandler := newEventsHandler()

	//logger.Info("ParseEventHandlers super:", superEventsHandler)
	if superEventsHandler != nil {
		for eventType, handler := range superEventsHandler.Handlers {
			if strings.HasPrefix(eventType, EvtUnitPrefix) {
				context.unitDef.EventsHandler.AddHandler(eventType, &CompEventHandler{CompDef: compDef, EventHandler: handler})
			} else {
				eventsHandler.AddHandler(eventType, handler)
			}
		}
	}

	//logger.Info("ParseEventHandlers def:", def)
	for eventType, eventPropsIf := range dataxform.SIMapGetByKeyAsMap(def, "eventHandlers") {
		eventProps := dataxform.AsSIMap(eventPropsIf)
		eventAction := dataxform.SIMapGetByKeyAsString(eventProps, "action")
		eventHandler := newEventHandler()
		eventHandler.JsCode = eventAction
		eventsHandler.AddHandler(eventType, eventHandler)
	}
	//logger.Info("eventsHandler created:", eventsHandler)
	return eventsHandler
}

func (comp *CompDef) ChildRefId() string {
	return comp.Props["cr"].(string)
}
