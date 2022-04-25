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
	fileNameRegex := dataxform.SIMapGetByKeyAsString(configData, "fileNameRegex")
	fileNameRegexCrToCompType := dataxform.SIMapGetByKeyAsBoolean(configData, "fileNameRegexCrToCompType")
	fileNameFromAutoProperty := dataxform.SIMapGetByKeyAsString(configData, "fileNameFromAutoProperty")
	fileName := dataxform.SIMapGetByKeyAsString(configData, "fileName")
	pathExpr := dataxform.SIMapGetByKeyAsString(configData, "pathExpr")
	pathNodes := dataxform.SIMapGetByKeyAsBoolean(configData, "pathNodes")
	filterExprList := dataxform.SIMapGetByKeyAsString(configData, "filterExpr")
	self := dataxform.SIMapGetByKeyAsBoolean(configData, "self")
	convertMapDescr := dataxform.SIMapGetByKeyAsString(configData, "convertMapDescr")

	var fileAsTemplatedYaml *loader.TemplatedYaml
	if self {
		fileAsTemplatedYaml = loader.ConvertTemplatedYaml([]byte(edContext.content), "self")
	} else {
		if fileNameRegex != "" {
			reFileName, err := regexp.Compile(fileNameRegex)
			if err != nil {
				logger.Error("invalid regex:", err.Error())
				return compl
			}
			matches := reFileName.FindAllStringSubmatch(yamlPos.valuePrefx, -1)
			if len(matches) == 0 {
				return compl
			}
			lastMatch := matches[len(matches)-1]
			if len(lastMatch) > 1 {
				fileName = lastMatch[1]
			} else {
				fileName = lastMatch[0]
			}
			logger.Info("regex fileName:" + fileName)
			if fileNameRegexCrToCompType {
				selfAsTemplatedYaml := loader.ConvertTemplatedYaml([]byte(edContext.content), "self")
				selfData := selfAsTemplatedYaml.IDef
				rePath, _ := convertYamlpathToRegex("[%].cr", yamlPos)
				dataxform.IApplyFn(selfData, func(entry dataxform.CollectionEntry) {
					logger.Info("sleaf:", entry.LinearKey())
					if rePath.MatchString(entry.LinearKey()) {
						logger.Info("match")
						if dataxform.IAsString(entry.Value()) == fileName {
							fileName = dataxform.IAsString(entry.SameLevelValue("compType"))
							logger.Info("comptype fileName:" + fileName)
							entry.Stop()
						}
					}
				})
			}
		}
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
					logger.Info("fileName:" + fileName)
				}
			})
		}
		if fileNameFromAutoProperty != "" {
			fileName = dataxform.SIMapGetByKeyAsString(configData, fileNameFromAutoProperty)
		}
		fileAsTemplatedYaml = loader.GetTemplatedYaml(fileName, "ide70/"+folderPrefix+"/")
	}
	if fileAsTemplatedYaml != nil {
		fileData := fileAsTemplatedYaml.IDef
		logger.Info("pathExpr:", pathExpr)
		rePath, selector := convertYamlpathToRegex(pathExpr, yamlPos)
		if rePath != nil {
			treeIterationFn := dataxform.IApplyFn
			if pathNodes {
				treeIterationFn = dataxform.IApplyFnToNodes
			}
			treeIterationFn(fileData, func(entry dataxform.CollectionEntry) {
				logger.Info("leaf:", entry.LinearKey())
				if rePath.MatchString(entry.LinearKey()) {
					value, descr := leafValDescr(entry, selector)
					logger.Info("match val:", value)
					logger.Info("match descr:", descr)

					filtered := false
					if filterExprList != "" {
						logger.Info("filterExprList:", filterExprList)
						filterExprArr := strings.Split(filterExprList, "|")
						for _, filterExpr := range filterExprArr {
							reFilterExpr, isFilterValue := convertYamlpathToRegex(filterExpr, yamlPos)
							logger.Info("filterExpr:", filterExpr)
							if reFilterExpr != nil {
								logger.Info("examine nodes")
								dataxform.IApplyFnToNodes(fileData, func(entry dataxform.CollectionEntry) {
									logger.Info("lin key:", entry.LinearKey())
									if reFilterExpr.MatchString(entry.LinearKey()) {
										filterValue, _ := leafValDescr(entry, isFilterValue)
										if value == filterValue {
											filtered = true
										}
									}
								})
								logger.Info("examine nodes done")
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
						actconfigData := dataxform.SIMapLightCopy(configData)
						if descr != "" {
							actconfigData["descr"] = descr
						}
						compl = addCompletion(value, edContext, actconfigData, compl)
					}
				}
			})
			logger.Info("IApplyFn finished")
		}

	}

	return compl
}

func leafValDescr(entry dataxform.CollectionEntry, selector string) (string, string) {
	switch selector {
	case "value":
		return dataxform.IAsString(entry.Value()), ""
	case "fullKey":
		return entry.LinearKey(), valueToDescr(entry.Value())
	}
	return entry.Key(), valueToDescr(entry.Value())
}

func valueToDescr(value interface{}) string {
	valueStr := dataxform.IAsString(value)
	if valueStr != "" {
		return valueStr
	}
	return dataxform.SIMapGetByKeyAsString(dataxform.IAsSIMap(value), "descr")
}

func convertYamlpathToRegex(path string, ypos *YamlPosition) (*regexp.Regexp, string) {
	pathTokens := strings.Split(path, ":")
	selector := "key"
	if len(pathTokens) > 1 {
		selector = pathTokens[1]
	}
	path = pathTokens[0]
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
			return nil, ""
		}
		path = strings.Join(absPathTokens[:len(absPathTokens)-levelBack], ".") + "." + path
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
		return nil, ""
	}
	return re, selector
}
