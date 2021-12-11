package codecomplete

import (
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
	"regexp"
	"strings"
)

// yamlPath expression
// key1.*

func yamlPathCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	folderPrefix := dataxform.SIMapGetByKeyAsString(configData, "folderPrefix")
	fileNameExpr := dataxform.SIMapGetByKeyAsString(configData, "fileNameExpr")
	fileName := dataxform.SIMapGetByKeyAsString(configData, "fileName")
	pathExpr := dataxform.SIMapGetByKeyAsString(configData, "pathExpr")
	filterExprList := dataxform.SIMapGetByKeyAsString(configData, "filterExpr")
	self := dataxform.SIMapGetByKeyAsBoolean(configData, "self")
	convertMapDescr := dataxform.SIMapGetByKeyAsString(configData, "convertMapDescr")

	var fileAsTemplatedYaml *loader.TemplatedYaml
	if self {
		fileAsTemplatedYaml = loader.ConvertTemplatedYaml([]byte(edContext.content), "self")
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
					fileName = dataxform.IAsString(entry.Value())
					logger.Info("fileName:"+ fileName)
				}
			})
		}
		fileAsTemplatedYaml = loader.GetTemplatedYaml(fileName, "ide70/"+folderPrefix+"/")
	}
	if fileAsTemplatedYaml != nil {
		fileData := fileAsTemplatedYaml.IDef
		logger.Info("pathExpr:", pathExpr)
		rePath, isValue := convertYamlpathToRegex(pathExpr, yamlPos)
		if rePath != nil {
			dataxform.IApplyFn(fileData, func(entry dataxform.CollectionEntry) {
				logger.Info("leaf:", entry.LinearKey())
				if rePath.MatchString(entry.LinearKey()) {
					logger.Info("match")
					value := leafVal(entry, isValue)

					filtered := false
					if filterExprList != "" {
						logger.Info("filterExpr:", filterExprList)
						filterExprArr := strings.Split(filterExprList, "|")
						for _, filterExpr := range filterExprArr {
							reFilterExpr, isFilterValue := convertYamlpathToRegex(filterExpr, yamlPos)
							if reFilterExpr != nil {
								dataxform.IApplyFnToNodes(fileData, func(entry dataxform.CollectionEntry) {
									if reFilterExpr.MatchString(entry.LinearKey()) {
										filterValue := leafVal(entry, isFilterValue)
										if value == filterValue {
											filtered = true
										}
									}
								})
							}
						}
					}

					logger.Info("kpfx:", yamlPos.keyPrefix)
					if convertMapDescr != "" && yamlPos.keyPrefix == value && !yamlPos.keyComplete {
						matchConfigData := map[string]interface{}{"singleToMap": true, "mapHead": true, "descr": convertMapDescr}
						compl = addCompletion(value, edContext, matchConfigData, compl)
					} else {
						if filtered {
							logger.Info("filtered:", value)
							return
						}
						compl = addCompletion(value, edContext, configData, compl)
					}
				}
			})
			logger.Info("IApplyFn finished")
		}

	}

	return compl
}

func leafVal(entry dataxform.CollectionEntry, isValue bool) string {
	if isValue {
		return dataxform.IAsString(entry.Value())
	} else {
		return entry.Key()
	}
}

func convertYamlpathToRegex(path string, ypos *YamlPosition) (*regexp.Regexp, bool) {
	levelBack := 0
	for strings.HasPrefix(path, "../") {
		path = strings.TrimPrefix(path, "../")
		levelBack++
	}
	if strings.HasPrefix(path, ".") {
		path = ypos.getIndexedKey() + path
	}
	if levelBack > 0 {
		logger.Info("indexed key:" + ypos.getIndexedKey())
		absPathTokens := strings.Split(ypos.getIndexedKey(), ".")
		if len(absPathTokens) < levelBack {
			logger.Error("relative expr failed")
			return nil, false
		}
		path = strings.Join(absPathTokens[:len(absPathTokens)-levelBack], ".") + "." + path
	}
	isValue := strings.HasSuffix(path, ":value")
	if isValue {
		path = strings.TrimSuffix(path, ":value")
	}
	path = strings.ReplaceAll(path, "%", "\\w+")
	path = strings.ReplaceAll(path, "*", ".*")
	path = strings.ReplaceAll(path, "[", "\\[")
	path = strings.ReplaceAll(path, "]", "\\]")
	path = strings.ReplaceAll(path, ".", "\\.")
	logger.Info("regex:", path)
	re, err := regexp.Compile(path)
	if err != nil {
		logger.Error("compiling regex:", err.Error())
		return nil, false
	}
	return re, isValue
}
