package server

import (
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
)

func yamlDataCompleter(yamlPos *YamlPosition, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	compl = append(compl, newCompletion("yamlDataCompleter", "yamlDataCompleter", "yamlDataCompleter"))
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
		for k, v := range rootEntry {
			descr := dataxform.IAsString(v)
			compl = append(compl, newCompletion(k+": ", k, descr))
		}
	}

	return compl
}
