package codecomplete

import (
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/loader"
)

func dictCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	dictName := api.SIMapGetByKeyAsString(configData, "dictName")
	valueMode := api.SIMapGetByKeyAsBoolean(configData, "value")
	fileAsTemplatedYaml := loader.GetTemplatedYaml(dictName, "ide70/dcfg/dict/")
	if fileAsTemplatedYaml != nil {
		fileData := fileAsTemplatedYaml.Def
		itemList := api.SIMapGetByKeyAsList(fileData, "items")
		for _, itemIf := range itemList {
			itemData := api.IAsSIMap(itemIf)
			code := api.SIMapGetByKeyAsString(itemData, "code")
			descr := api.SIMapGetByKeyAsString(itemData, "descr")
			if valueMode {
				compl = addValueCompletion(code, descr, edContext, configData, compl)
			} else {
				configDataTmp := api.SIMapLightCopy(configData)
				configDataTmp["descr"] = descr
				compl = addCompletion(code, edContext, configDataTmp, compl)
			}
		}
	}

	return compl
}
