package codecomplete

import (
	//"fmt"
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
)

func templateCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	code := yamlPos.valuePrefx
	logger.Info("code:", code+"|")

	selfAsTemplatedYaml := loader.ConvertTemplatedYaml([]byte(edContext.content), "self")
	selfData := selfAsTemplatedYaml.Def
	unitInterfaceData := dataxform.SIMapGetByKeyAsMap(selfData, "unitInterface")
	ppData := dataxform.SIMapGetByKeyAsMap(selfData, "privateProperties")
	propertyData := dataxform.SIMapGetByKeyAsMap(unitInterfaceData, "properties")
	propMap := map[string]interface{}{}
	procPropertyData(ppData, propMap)
	procPropertyData(propertyData, propMap)
	propMap["sid"] = "Components unique ID"
	
	filterData := dataxform.SIMapGetByKeyAsMap(configData, "filters")
	
	for name, propDataIf := range propMap {
		propData := dataxform.IAsSIMap(propDataIf)
		filterMatch := true
		for key, vIf := range filterData {
			value := dataxform.SIMapGetByKeyAsString(propData, key)
			if value != dataxform.IAsString(vIf) {
				filterMatch = false
				break
			}
		}
		if !filterMatch {
			continue
		}
		
		propDescr := dataxform.SIMapGetByKeyAsString(propData, "descr")
		compl = addValueCompletion("."+name, propDescr, edContext, configData, compl)
		//compl = append(compl, newCompletion("."+name, "."+name, propDescr))
	}


	fileAsTemplatedYaml := loader.GetTemplatedYaml("templateComplete", "ide70/dcfg/")
	if fileAsTemplatedYaml == nil {
		return compl
	}
	templateConfig := fileAsTemplatedYaml.Def
	methodsMap := dataxform.SIMapGetByKeyAsMap(templateConfig, "methods")
	for methodName, methodDataIf := range methodsMap {
		methodData := dataxform.IAsSIMap(methodDataIf)

		filterMatch := true
		for key, vIf := range filterData {
			value := dataxform.SIMapGetByKeyAsString(methodData, key)
			if value != dataxform.IAsString(vIf) {
				filterMatch = false
				break
			}
		}
		if !filterMatch {
			continue
		}

		//wrapTag := dataxform.SIMapGetByKeyAsString(methodData, "wrapTag")
		methodDescr := dataxform.SIMapGetByKeyAsString(methodData, "descr")

		//compl = append(compl, newCompletion(methodName, methodName, methodDescr))
		compl = addValueCompletion(methodName, methodDescr, edContext, configData, compl)
	}

	return compl
}

func procPropertyData(propertyData map[string]interface{}, res map[string]interface{}) {
	for name, data := range propertyData {
		switch dt := data.(type) {
		case string:
			entry := map[string]interface{}{}
			entry["descr"] = dt
			res[name] = entry
		case map[string]interface{}:
			res[name] = data
		}
	}
}
