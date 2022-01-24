package codecomplete

import (
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
)

func dictCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	dictName := dataxform.SIMapGetByKeyAsString(configData, "dictName")
	fileAsTemplatedYaml := loader.GetTemplatedYaml(dictName, "ide70/dcfg/dict/")
	if fileAsTemplatedYaml != nil {
		fileData := fileAsTemplatedYaml.Def
		itemList := dataxform.SIMapGetByKeyAsList(fileData, "items")
		for _, itemIf := range itemList {
			itemData := dataxform.IAsSIMap(itemIf)
			code := dataxform.SIMapGetByKeyAsString(itemData, "code")
			descr := dataxform.SIMapGetByKeyAsString(itemData, "descr")
			compl = addValueCompletion(code, descr, edContext, configData, compl)
		}
	}

	return compl
}
