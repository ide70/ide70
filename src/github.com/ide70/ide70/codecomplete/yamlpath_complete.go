package codecomplete

import (
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/loader"
	"regexp"
	"strings"
)

// yamlPath expression
// key1.*

func yamlPathCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	folderPrefix := api.SIMapGetByKeyAsString(configData, "folderPrefix")
	fileNameExpr := api.SIMapGetByKeyAsString(configData, "fileNameExpr")
	fileNameRegex := api.SIMapGetByKeyAsString(configData, "fileNameRegex")
	fileNameRegexCrToCompType := api.SIMapGetByKeyAsBoolean(configData, "fileNameRegexCrToCompType")
	fileNameFromAutoProperty := api.SIMapGetByKeyAsString(configData, "fileNameFromAutoProperty")
	fileName := api.SIMapGetByKeyAsString(configData, "fileName")
	pathExpr := api.SIMapGetByKeyAsString(configData, "pathExpr")
	pathNodes := api.SIMapGetByKeyAsBoolean(configData, "pathNodes")
	filterExprList := api.SIMapGetByKeyAsString(configData, "filterExpr")
	self := api.SIMapGetByKeyAsBoolean(configData, "self")
	convertMapDescr := api.SIMapGetByKeyAsString(configData, "convertMapDescr")

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
				api.IApplyFn(selfData, func(entry api.CollectionEntry) {
					logger.Info("sleaf:", entry.LinearKey())
					if rePath.MatchString(entry.LinearKey()) {
						logger.Info("match")
						if api.IAsString(entry.Value()) == fileName {
							fileName = api.IAsString(entry.SameLevelValue("compType"))
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
			api.IApplyFn(selfData, func(entry api.CollectionEntry) {
				logger.Info("sleaf:", entry.LinearKey())
				if rePath.MatchString(entry.LinearKey()) {
					logger.Info("match")
					fileName = api.IAsString(entry.Value())
					logger.Info("fileName:" + fileName)
				}
			})
		}
		if fileNameFromAutoProperty != "" {
			fileName = api.SIMapGetByKeyAsString(configData, fileNameFromAutoProperty)
		}
		fileAsTemplatedYaml = loader.GetTemplatedYaml(fileName, "ide70/"+folderPrefix+"/")
	}
	if fileAsTemplatedYaml != nil {
		fileData := fileAsTemplatedYaml.IDef
		logger.Info("pathExpr:", pathExpr)
		rePath, selector := convertYamlpathToRegex(pathExpr, yamlPos)
		if rePath != nil {
			treeIterationFn := api.IApplyFn
			if pathNodes {
				treeIterationFn = api.IApplyFnToNodes
			}
			treeIterationFn(fileData, func(entry api.CollectionEntry) {
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
								api.IApplyFnToNodes(fileData, func(entry api.CollectionEntry) {
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
						actconfigData := api.SIMapLightCopy(configData)
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

func leafValDescr(entry api.CollectionEntry, selector string) (string, string) {
	switch selector {
	case "value":
		return api.IAsString(entry.Value()), ""
	case "fullKey":
		return entry.LinearKey(), valueToDescr(entry.Value())
	}
	return entry.Key(), valueToDescr(entry.Value())
}

func valueToDescr(value interface{}) string {
	valueStr := api.IAsString(value)
	if valueStr != "" {
		return valueStr
	}
	return api.SIMapGetByKeyAsString(api.IAsSIMap(value), "descr")
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
