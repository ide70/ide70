package comp

import (
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/loader"
)

// a component instance
type CompDef struct {
	CompType      *CompType
	Children      []*CompDef
	Props         map[string]interface{}
	EventsHandler *CompDefEventsHandler
	Triggers      map[string]string
}

func ParseCompDef(def map[string]interface{}, context *UnitDefContext) *CompDef {
	logger.Debug("ParseCompDef", def)
	compDef := &CompDef{}
	compTypeName := def["compType"].(string)
	compDef.CompType = GetCompType(compTypeName, context.appParams)
	compDef.Props = def

	// TODO: lista merge nem az igazi
	accDefCopy := api.IAsSIMap(api.SIMapCopy(compDef.CompType.AccessibleDef))
	api.SIMapInjectDefaults(accDefCopy, compDef.Props)

	logger.Debug("ParseCompDef id before")
	if def["cr"] == nil {
		def["cr"] = context.getNextId(compTypeName)
	}
	logger.Debug("ParseCompDef id", def["cr"])

	//compDef.EventsHandler = ParseEventHandlers(def, compDef.CompType.EventsHandler, context, compDef)

	parseExternalReferences(def, context, compDef)
	parseCalcs(def, context, compDef)
	parseTriggerDefs(def, compDef)
	parseUnitRelatedDefs(def, context)

	logger.Debug("ParseCompDef end")

	return compDef
}

func (compDef *CompDef) ParseEventHandlers(context *UnitDefContext) {
	compDef.EventsHandler = ParseEventHandlers(compDef.Props, compDef.CompType.EventsHandler, context, compDef)
}


func parseTriggerDefs(def map[string]interface{}, compDef *CompDef) {
	compDef.Triggers = map[string]string{}
	setTriggers := api.SIMapGetByKeyAsMap(def, "setTriggers")
	for propertyKey, eventTypeIf := range setTriggers {
		compDef.Triggers[propertyKey] = api.IAsString(eventTypeIf)
	}
}

func parseUnitRelatedDefs(def map[string]interface{}, context *UnitDefContext) {
	unitRelatedMap := api.SIMapGetByKeyAsMap(def, "injectToUnit")
	if len(unitRelatedMap) > 0 {
		api.SIMapInjectDefaults(unitRelatedMap, context.unitDef.Props)
	}
	copyList := api.SIMapGetByKeyAsList(def, "copyPropertyToUnit")
	for _, copyItemIf := range copyList {
		copyitem := api.AsSIMap(copyItemIf)
		srcProp := api.SIMapGetByKeyAsString(copyitem, "srcProp")
		dstProp := api.SIMapGetByKeyAsString(copyitem, "dstProp")
		context.unitDef.Props[dstProp] = def[srcProp]
		logger.Debug("unit poperty:", dstProp, context.unitDef.Props[dstProp])
	}
}

func parseCalcs(def map[string]interface{}, context *UnitDefContext, compDef *CompDef) {
	api.IApplyFnToNodes(def, func(entry api.CollectionEntry) {
		if entry.Key() == "calc" {
			value := api.IAsString(entry.Value())
			if value != "" {
				calc := &Calc{Comp: compDef, PropertyKey: entry.Parent().Key(), jsCode: value}
				logger.Debug("calc added:", *calc)
				context.unitDef.CalcArr = append(context.unitDef.CalcArr, calc)
			}
		}
	})
}

func parseExternalReferences(def map[string]interface{}, context *UnitDefContext, compDef *CompDef) {
	api.IApplyFnToNodes(def, func(entry api.CollectionEntry) {
		if entry.Key() == "externalReference" {
			props := api.IAsSIMap(entry.Value())
			if len(props) != 0 {
				fileName := api.SIMapGetByKeyAsString(props, "fileName")
				key := api.SIMapGetByKeyAsString(props, "key")
				// betölteni
				extConfig := loader.GetTemplatedYaml(fileName, "ide70/dcfg/")
				value := api.SIMapGetByKeyChain(extConfig.Def, key)
				logger.Debug("external definition:", key, value)
				entry.Parent().Update(value)
			}
		}
	})
}

func ParseEventHandlers(def map[string]interface{}, superEventsHandler *CompDefEventsHandler, context *UnitDefContext, compDef *CompDef) *CompDefEventsHandler {
	eventsHandler := newEventsHandler()

	if compDef != nil {
		logger.Debug("processing events for:", compDef.ChildRefId())
	}
	//logger.Debug("ParseEventHandlers super:", superEventsHandler)
	var initEventCodeList map[string]bool
	
	if context != nil {
		codeList := context.unitDef.getInitialEventCodes()
		codeList = append(codeList, context.unitDef.getPostRenderEventCodes()...)
		initEventCodeList = api.StringListToSet(codeList)
		
	} else {
		initEventCodeList = map[string]bool{}
	}
	
	eventHandlers := api.SIMapGetByKeyAsMap(def, "eventHandlers")
	logger.Debug("ehs:", eventHandlers)
	if superEventsHandler != nil {
		for eventType, handler := range superEventsHandler.Handlers {
			if initEventCodeList[eventType] {
				logger.Debug("add super event to unit:"+eventType )
				context.unitDef.EventsHandler.AddHandler(eventType, &CompEventHandler{CompDef: compDef, EventHandler: handler})
			}
			logger.Debug("add super event to comp:",eventType, " " ,len(handler.JsCode) )
			eventsHandler.AddHandler(eventType, handler)
		}
	}

	//logger.Debug("ParseEventHandlers def:", def)
	for eventType, eventPropsIf := range eventHandlers {
		eventProps := api.AsSIMap(eventPropsIf)
		eventAction := api.SIMapGetByKeyAsString(eventProps, "action")
		eventHandler := newEventHandler()
		eventHandler.JsCode = eventAction
		eventHandler.PropertyKey = api.SIMapGetByKeyAsString(eventProps, "propertyKey")
		if initEventCodeList[eventType] {
			logger.Debug("add event to unit:"+eventType )
			context.unitDef.EventsHandler.AddHandler(eventType, &CompEventHandler{CompDef: compDef, EventHandler: eventHandler})
		}
		logger.Debug("add event to comp:",eventType , " " ,len(eventAction) )
		eventsHandler.AddHandler(eventType, eventHandler)
	}
	logger.Debug("eventsHandler created:")
	for k,v:= range eventsHandler.Handlers {
		logger.Debug("evhan:", k, len(v.JsCode))
	}
	return eventsHandler
}

func (comp *CompDef) ChildRefId() string {
	return comp.Props["cr"].(string)
}
