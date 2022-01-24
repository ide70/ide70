package codecomplete

import (
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
)

func yamlDataCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	folderPrefix := dataxform.SIMapGetByKeyAsString(configData, "folderPrefix")
	fileNameSrc := dataxform.SIMapGetByKeyAsString(configData, "fileNameSrc")
	rootKey := dataxform.SIMapGetByKeyAsString(configData, "rootKey")
	fileName := ""
	if fileNameSrc == "yamlParentValue" {
		fileName = yamlPos.parent.valuePrefx
	} else {
		fileName = fileNameSrc 
	}
	fileAsTemplatedYaml := loader.GetTemplatedYaml(fileName, "ide70/"+folderPrefix+"/")
	if fileAsTemplatedYaml != nil {
		fileData := fileAsTemplatedYaml.Def
		rootEntry := dataxform.SIMapGetByKeyChainAsMap(fileData, rootKey)
		if len(rootEntry) > 0 {
			compl = completerCore(yamlPos, edContext, rootEntry, compl)
		}
	}

	return compl
}
