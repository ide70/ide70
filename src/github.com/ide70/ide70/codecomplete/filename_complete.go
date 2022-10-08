package codecomplete

import (
	//"fmt"
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/loader"
	"github.com/ide70/ide70/util/file"
)

func fileNameCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	folderPrefix := api.SIMapGetByKeyAsString(configData, "folderPrefix")
	trimSuffix := api.SIMapGetByKeyAsString(configData, "trimSuffix")
	foldersOnly := api.SIMapGetByKeyAsBoolean(configData, "foldersOnly")
	fileNames := []string{}
	if foldersOnly {
		fileNames = file.DirList("ide70/"+folderPrefix, "ide70/"+folderPrefix+"/")
	} else {
		fileNames = file.FileList("ide70/"+folderPrefix, "ide70/"+folderPrefix+"/", trimSuffix)
	}
	for _, fileName := range fileNames {
		componentDescr := ""
		complConfigData := api.SIMapLightCopy(configData)
		if trimSuffix == ".yaml" {
			fileAsTemplatedYaml := loader.GetTemplatedYaml(fileName, "ide70/"+folderPrefix+"/")
			if fileAsTemplatedYaml != nil {
				fileData := fileAsTemplatedYaml.Def
				fileInterface := api.SIMapGetByKeyAsMap(fileData, "unitInterface")
				componentDescr = api.SIMapGetByKeyAsString(fileInterface, "descr")
				complConfigData["subProperties"] = getMandatoryProperties(fileInterface)
			}
		}
		complConfigData["descr"] = componentDescr
		compl = addCompletion(fileName, edContext, complConfigData, compl)
	}
	return compl
}

func getMandatoryProperties(fileInterface map[string]interface{}) []interface{} {
	keys := []interface{}{}
	properties := api.SIMapGetByKeyAsMap(fileInterface, "properties")
	for propKey, propAttrsIf := range properties {
		switch propAttrs := propAttrsIf.(type) {
		case map[string]interface{}:
			if api.SIMapGetByKeyAsBoolean(propAttrs, "mandatory") {
				keys = append(keys, propKey)
			}
		}
	}
	return keys
}
