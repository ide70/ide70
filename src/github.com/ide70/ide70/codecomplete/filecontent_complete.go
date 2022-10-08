package codecomplete

import (
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/loader"
	"regexp"
)

// `([\.#][_A-Za-z0-9\-]+)[^}]*{`

func fileContentCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	folderPrefix := api.SIMapGetByKeyAsString(configData, "folderPrefix")
	fileNameExpr := api.SIMapGetByKeyAsString(configData, "fileNameExpr")
	searchExpr := api.SIMapGetByKeyAsString(configData, "searchExpr")
	self := api.SIMapGetByKeyAsBoolean(configData, "self")

	reSearch, err := regexp.Compile(searchExpr)
	if err != nil {
		logger.Error("error compiling regex: " + searchExpr)
		return compl
	}

	if self {
		fileContents := edContext.content
		compl = scanContents(fileContents, reSearch, compl)
	} else {
		if fileNameExpr != "" {
			selfAsTemplatedYaml := loader.ConvertTemplatedYaml([]byte(edContext.content), "self")
			selfData := selfAsTemplatedYaml.IDef
			logger.Debug("fileNameExpr:", fileNameExpr)
			rePath, _ := convertYamlpathToRegex(fileNameExpr, yamlPos)
			api.IApplyFn(selfData, func(entry api.CollectionEntry) {
				logger.Debug("sleaf:", entry.LinearKey())
				if rePath.MatchString(entry.LinearKey()) {
					logger.Debug("match")
					fileName := api.IAsString(entry.Value())
					logger.Debug("fileName:" + fileName)
					fileContents := loader.LoadFileContents(fileName, "ide70/"+folderPrefix+"/")
					compl = scanContents(fileContents, reSearch, compl)
				}
			})
		}

	}

	return compl
}

func scanContents(fileContents string, reSearch *regexp.Regexp, compl []map[string]string) []map[string]string {
	matches := reSearch.FindAllStringSubmatch(fileContents, -1)
	for _, match := range matches {
		descr := ""
		match1 := match[0]
		if len(match) > 1 {
			match1 = match[1]
			if len(match) > 2 {
				descr = match[2]
			}
		}
		compl = append(compl, newCompletion(match1, match1, descr))
	}

	return compl
}
