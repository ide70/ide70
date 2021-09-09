package codecomplete

import (
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
)

func yamlDataCompleter(yamlPos *YamlPosition, col int, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	folderPrefix := dataxform.SIMapGetByKeyAsString(configData, "folderPrefix")
	fileNameSrc := dataxform.SIMapGetByKeyAsString(configData, "fileNameSrc")
	rootKey := dataxform.SIMapGetByKeyAsString(configData, "rootKey")
	fileName := ""
	if fileNameSrc == "yamlParentValue" {
		fileName = yamlPos.parent.valuePrefx
	}
	fileAsTemplatedYaml := loader.GetTemplatedYaml(fileName, "ide70/"+folderPrefix+"/")
	if fileAsTemplatedYaml != nil {
		fileData := fileAsTemplatedYaml.Def
		rootEntry := dataxform.SIMapGetByKeyChainAsMap(fileData, rootKey)
		if len(rootEntry) > 0 {
			compl = completerCore(yamlPos, col, rootEntry, compl)
		}
	}

	return compl
}
