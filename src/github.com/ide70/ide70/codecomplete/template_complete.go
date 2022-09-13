package codecomplete

import (
	//"fmt"
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/loader"
)

func templateCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	code := yamlPos.valuePrefx
	logger.Info("code:", code+"|")

	selfAsTemplatedYaml := loader.ConvertTemplatedYaml([]byte(edContext.content), "self")
	selfData := selfAsTemplatedYaml.Def
	unitInterfaceData := api.SIMapGetByKeyAsMap(selfData, "unitInterface")
	ppData := api.SIMapGetByKeyAsMap(selfData, "privateProperties")
	propertyData := api.SIMapGetByKeyAsMap(unitInterfaceData, "properties")
	propMap := map[string]interface{}{}
	procPropertyData(ppData, propMap)
	procPropertyData(propertyData, propMap)
	propMap["sid"] = "Components unique ID"
	propMap["Children"] = map[string]interface{}{"descr":"Child components", "type": "array"}
	
	filterData := api.SIMapGetByKeyAsMap(configData, "filters")
	
	for name, propDataIf := range propMap {
		propData := api.IAsSIMap(propDataIf)
		filterMatch := true
		for key, vIf := range filterData {
			value := api.SIMapGetByKeyAsString(propData, key)
			if value != api.IAsString(vIf) {
				filterMatch = false
				break
			}
		}
		if !filterMatch {
			continue
		}
		
		propDescr := api.SIMapGetByKeyAsString(propData, "descr")
		compl = addValueCompletion("$."+name, propDescr, edContext, configData, compl)
		//compl = append(compl, newCompletion("."+name, "."+name, propDescr))
	}


	fileAsTemplatedYaml := loader.GetTemplatedYaml("templateComplete", "ide70/dcfg/")
	if fileAsTemplatedYaml == nil {
		return compl
	}
	templateConfig := fileAsTemplatedYaml.Def
	methodsMap := api.SIMapGetByKeyAsMap(templateConfig, "methods")
	for methodName, methodDataIf := range methodsMap {
		methodData := api.IAsSIMap(methodDataIf)

		filterMatch := true
		for key, vIf := range filterData {
			value := api.SIMapGetByKeyAsString(methodData, key)
			if value != api.IAsString(vIf) {
				filterMatch = false
				break
			}
		}
		if !filterMatch {
			continue
		}

		//wrapTag := api.SIMapGetByKeyAsString(methodData, "wrapTag")
		methodDescr := api.SIMapGetByKeyAsString(methodData, "descr")

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
