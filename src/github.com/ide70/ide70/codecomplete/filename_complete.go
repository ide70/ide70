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
	fileNames := file.FileList("ide70/"+folderPrefix, "ide70/"+folderPrefix+"/", trimSuffix)
	for _, fileName := range fileNames {
		fileAsTemplatedYaml := loader.GetTemplatedYaml(fileName, "ide70/"+folderPrefix+"/")
		componentDescr := ""
		if fileAsTemplatedYaml != nil {
			fileData := fileAsTemplatedYaml.Def
			fileInterface := dataxform.SIMapGetByKeyAsMap(fileData, "unitInterface")
			componentDescr = dataxform.SIMapGetByKeyAsString(fileInterface, "descr")
		}
		complConfigData := dataxform.SIMapLightCopy(configData)
		complConfigData["descr"] = componentDescr
		compl = addCompletion(fileName, edContext, complConfigData, compl)
		//compl = append(compl, newCompletion(fileName, fileName, componentDescr))

	}
	return compl
}
