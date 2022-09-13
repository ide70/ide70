package codecomplete

import (
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/loader"
)

func yamlDataCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	folderPrefix := api.SIMapGetByKeyAsString(configData, "folderPrefix")
	fileNameSrc := api.SIMapGetByKeyAsString(configData, "fileNameSrc")
	rootKey := api.SIMapGetByKeyAsString(configData, "rootKey")
	fileName := ""
	if fileNameSrc == "yamlParentValue" {
		fileName = yamlPos.parent.valuePrefx
	} else {
		fileName = fileNameSrc 
	}
	fileAsTemplatedYaml := loader.GetTemplatedYaml(fileName, "ide70/"+folderPrefix+"/")
	if fileAsTemplatedYaml != nil {
		fileData := fileAsTemplatedYaml.Def
		rootEntry := api.SIMapGetByKeyChainAsMap(fileData, rootKey)
		if len(rootEntry) > 0 {
			compl = completerCore(yamlPos, edContext, rootEntry, compl)
		}
	}

	return compl
}
