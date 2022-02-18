package codecomplete

import (
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
	"regexp"
)

// `([\.#][_A-Za-z0-9\-]+)[^}]*{`

func fileContentCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	folderPrefix := dataxform.SIMapGetByKeyAsString(configData, "folderPrefix")
	fileNameExpr := dataxform.SIMapGetByKeyAsString(configData, "fileNameExpr")
	searchExpr := dataxform.SIMapGetByKeyAsString(configData, "searchExpr")
	self := dataxform.SIMapGetByKeyAsBoolean(configData, "self")

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
			logger.Info("fileNameExpr:", fileNameExpr)
			rePath, _ := convertYamlpathToRegex(fileNameExpr, yamlPos)
			dataxform.IApplyFn(selfData, func(entry dataxform.CollectionEntry) {
				logger.Info("sleaf:", entry.LinearKey())
				if rePath.MatchString(entry.LinearKey()) {
					logger.Info("match")
					fileName := dataxform.IAsString(entry.Value())
					logger.Info("fileName:" + fileName)
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
