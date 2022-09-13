package codecomplete

import (
	//"fmt"
	"github.com/ide70/ide70/api"
)

func unionCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	logger.Info("unionCompleter configData:", configData)
	subCompleters := api.SIMapGetByKeyAsList(configData, "paramsList")
	completerType := api.SIMapGetByKeyAsString(configData, "completerType")
	descrPostfix := api.SIMapGetByKeyAsString(configData, "descrPostfix")
	firstNonemptyOnly := api.SIMapGetByKeyAsBoolean(configData, "firstNonemptyOnly")
	completerKey := completerType + "Completer"
	for _,subCompleterIf := range subCompleters {
		topLevelConfig := map[string]interface{}{}
		topLevelConfig[completerKey] = subCompleterIf
		completer, subConfigData := lookupCompleter(completerType, topLevelConfig)
			if completer != nil {
				subConfigData["descrPostfix"] = descrPostfix
				subConfigData["firstConst"] = configData["firstConst"]
				subConfigData["table"] = configData["table"]
				res := completer(yamlPos, edContext, subConfigData, compl)
				compl = append(compl, res...)
				logger.Info("len(res):", len(res))
				if len(res)>0 && firstNonemptyOnly {
					break
				}
			}
	}

	return compl

}
