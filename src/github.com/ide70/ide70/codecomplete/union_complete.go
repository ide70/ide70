package codecomplete

import (
	//"fmt"
	"github.com/ide70/ide70/dataxform"
)

func unionCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	logger.Info("unionCompleter configData:", configData)
	subCompleters := dataxform.SIMapGetByKeyAsList(configData, "paramsList")
	completerType := dataxform.SIMapGetByKeyAsString(configData, "completerType")
	descrPostfix := dataxform.SIMapGetByKeyAsString(configData, "descrPostfix")
	firstNonemptyOnly := dataxform.SIMapGetByKeyAsBoolean(configData, "firstNonemptyOnly")
	completerKey := completerType + "Completer"
	for _,subCompleterIf := range subCompleters {
		topLevelConfig := map[string]interface{}{}
		topLevelConfig[completerKey] = subCompleterIf
		completer, subConfigData := lookupCompleter(completerType, topLevelConfig)
			if completer != nil {
				subConfigData["descrPostfix"] = descrPostfix
				subConfigData["firstConst"] = configData["firstConst"]
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
