package codecomplete

import (
	//"fmt"
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
	"github.com/ide70/ide70/util/file"
)

func fileNameCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	folderPrefix := dataxform.SIMapGetByKeyAsString(configData, "folderPrefix")
	trimSuffix := dataxform.SIMapGetByKeyAsString(configData, "trimSuffix")
	foldersOnly := dataxform.SIMapGetByKeyAsBoolean(configData, "foldersOnly")
	fileNames := []string{}
	if foldersOnly {
		fileNames = file.DirList("ide70/"+folderPrefix, "ide70/"+folderPrefix+"/")
	} else {
		fileNames = file.FileList("ide70/"+folderPrefix, "ide70/"+folderPrefix+"/", trimSuffix)
	}
	for _, fileName := range fileNames {
		fileAsTemplatedYaml := loader.GetTemplatedYaml(fileName, "ide70/"+folderPrefix+"/")
		componentDescr := ""
		complConfigData := dataxform.SIMapLightCopy(configData)
		if fileAsTemplatedYaml != nil {
			fileData := fileAsTemplatedYaml.Def
			fileInterface := dataxform.SIMapGetByKeyAsMap(fileData, "unitInterface")
			componentDescr = dataxform.SIMapGetByKeyAsString(fileInterface, "descr")
			complConfigData["subProperties"] = getMandatoryProperties(fileInterface)
		}
		complConfigData["descr"] = componentDescr
		compl = addCompletion(fileName, edContext, complConfigData, compl)
	}
	return compl
}

func getMandatoryProperties(fileInterface map[string]interface{}) []interface{} {
	keys := []interface{}{}
	properties := dataxform.SIMapGetByKeyAsMap(fileInterface, "properties")
	for propKey, propAttrsIf := range properties {
		switch propAttrs := propAttrsIf.(type) {
			case  map[string]interface{}:
			if dataxform.SIMapGetByKeyAsBoolean(propAttrs, "mandatory") {
				keys = append(keys, propKey)
			}
		}
	}
	return keys
}
