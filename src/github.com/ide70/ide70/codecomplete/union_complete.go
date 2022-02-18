package codecomplete

import (
	//"fmt"
	"github.com/ide70/ide70/dataxform"
)

func unionCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	subCompleters := dataxform.SIMapGetByKeyAsList(configData, "paramsList")
	completerType := dataxform.SIMapGetByKeyAsString(configData, "completerType")
	descrPostfix := dataxform.SIMapGetByKeyAsString(configData, "descrPostfix")
	completerKey := completerType + "Completer"
	for _,subCompleterIf := range subCompleters {
		topLevelConfig := map[string]interface{}{}
		topLevelConfig[completerKey] = subCompleterIf
		completer, configData := lookupCompleter(completerType, topLevelConfig)
			if completer != nil {
				configData["descrPostfix"] = descrPostfix
				compl = append(compl, completer(yamlPos, edContext, configData, compl)...)
			}
	}

	return compl

}
