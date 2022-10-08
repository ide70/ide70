package codecomplete

import (
	//"fmt"
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/loader"
	"regexp"
	"strings"
)

var reWord = regexp.MustCompile(`\w+`)

func htmlCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	code := yamlPos.valuePrefx
	logger.Debug("code:", code+"|")

	if strings.HasSuffix(code, "{") {
		compl = append(compl, effectiveLoopVars(code, true)...)

		selfAsTemplatedYaml := loader.ConvertTemplatedYaml([]byte(edContext.content), "self")
		selfData := selfAsTemplatedYaml.Def
		unitInterfaceData := api.SIMapGetByKeyAsMap(selfData, "unitInterface")
		ppData := api.SIMapGetByKeyAsMap(selfData, "privateProperties")
		propertyData := api.SIMapGetByKeyAsMap(unitInterfaceData, "properties")
		propMap := map[string]string{}
		processPropertyData(ppData, propMap)
		processPropertyData(propertyData, propMap)
		propMap["sid"] = "Components unique ID"
		for name, descr := range propMap {
			compl = append(compl, newCompletion("{."+name+"}}", "."+name, descr))
		}

		fileAsTemplatedYaml := loader.GetTemplatedYaml("templateComplete", "ide70/dcfg/")
		if fileAsTemplatedYaml == nil {
			return compl
		}
		templateConfig := fileAsTemplatedYaml.Def
		methods := api.SIMapGetByKeyAsMap(templateConfig, "methods")
		for methodName, methodDataIf := range methods {
			methodData := api.IAsSIMap(methodDataIf)
			closeTag := api.SIMapGetByKeyAsString(methodData, "closeTag")
			methodDescr := api.SIMapGetByKeyAsString(methodData, "descr")
			value := "{" + methodName + "}}"
			if closeTag != "" {
				value += "{{" + closeTag + "}}"
			}
			compl = append(compl, newCompletion(value, methodName, methodDescr))
		}

		return compl
	}

	templStartIdx := strings.LastIndex(code, "{{")
	templEndIdx := strings.LastIndex(code, "}}")

	if templStartIdx > templEndIdx {
		edContext.contextType = "template"
		methodCode := code[templStartIdx+2:]
		methodTokens := strings.Split(methodCode, " ")
		if len(methodTokens) < 1 {
			return compl
		}

		fileAsTemplatedYaml := loader.GetTemplatedYaml("templateComplete", "ide70/dcfg/")
		if fileAsTemplatedYaml == nil {
			return compl
		}
		templateConfig := fileAsTemplatedYaml.Def
		methods := api.SIMapGetByKeyAsMap(templateConfig, "methods")
		logger.Debug("meth listed")
		methodName := ""
		methodTokenIdx := 0
		for i := len(methodTokens) - 1; i >= 0; i-- {
			if methods[methodTokens[i]] != nil {
				methodName = methodTokens[i]
				methodTokenIdx = i
				break
			}
		}

		if methodName == "" {
			return compl
		}

		paramNo := len(methodTokens) - 2 - methodTokenIdx
		logger.Debug("meth:", methodName)
		logger.Debug("paramNo:", paramNo)

		methodData := api.SIMapGetByKeyAsMap(methods, methodName)

		methodParams := api.SIMapGetByKeyAsList(methodData, "params")
		if paramNo >= len(methodParams) {
			return compl
		}

		compl = append(compl, effectiveLoopVars(code, false)...)

		methodParam := methodParams[paramNo]
		methodParamData := api.IAsSIMap(methodParam)
		logger.Debug("meth param data:", methodParamData)
		fixedValue := api.SIMapGetByKeyAsString(methodParamData, "fixedValue")
		paramDescr := api.SIMapGetByKeyAsString(methodParamData, "descr")
		
		compl = append(compl, newHeader(paramDescr))
		if fixedValue != "" {
			compl = append(compl, newCompletion(fixedValue, fixedValue, paramDescr))
			return compl
		}
		completer, configData := lookupCompleter("value", methodParamData)
		if completer != nil {
			compl = completer(yamlPos, edContext, configData, compl)
			return compl
		}

	}

	fileAsTemplatedYaml := loader.GetTemplatedYaml("htmlCompleterConfig", "ide70/dcfg/")
	if fileAsTemplatedYaml == nil {
		return compl
	}
	htmlConfig := fileAsTemplatedYaml.Def

	idxTagStart := strings.LastIndex(code, "<")
	idxTagEnd := strings.LastIndex(code, ">")
	if idxTagStart > idxTagEnd {
		if strings.HasSuffix(code, " ") {
			tagName := reWord.FindString(code[idxTagStart+1:])
			compl = append(compl, completeAttr(tagName, htmlConfig)...)
			return compl
		}
		idxAttrValueStart := strings.LastIndex(code, "=\"")
		idxAttrValueEnd := strings.LastIndex(code, "\"")
		if idxAttrValueStart+1 == idxAttrValueEnd {
			idxAttrStart := strings.LastIndex(code, " ")
			searchAttrName := code[idxAttrStart+1 : idxAttrValueStart]
			attrs := api.SIMapGetByKeyAsList(htmlConfig, "attrs")
			for _, attrIf := range attrs {
				attrData := api.IAsSIMap(attrIf)
				attrName := api.SIMapGetByKeyAsString(attrData, "name")
				if searchAttrName != attrName {
					continue
				}

				completer, configData := lookupCompleter("value", attrData)
				if completer != nil {
					compl = completer(yamlPos, edContext, configData, compl)
					return compl
				}
				break
			}
			return compl
		}
		return compl
	}

	tags := api.SIMapGetByKeyAsList(htmlConfig, "tags")
	for _, tagIf := range tags {
		tagData := api.IAsSIMap(tagIf)
		tagName := api.SIMapGetByKeyAsString(tagData, "name")
		tagDescr := api.SIMapGetByKeyAsString(tagData, "descr")
		tagCaption := "<" + tagName + ">"
		tagValue := tagCaption + "</" + tagName + ">"
		compl = append(compl, newCompletion(tagValue, tagCaption, tagDescr))
	}
	return compl
}

func processPropertyData(propertyData map[string]interface{}, res map[string]string) {
	for name, data := range propertyData {
		switch dt := data.(type) {
		case string:
			res[name] = dt
		case map[string]interface{}:
			res[name] = api.SIMapGetByKeyAsString(dt, "descr")
		}
	}
}

func completeAttr(tagName string, htmlConfig map[string]interface{}) []map[string]string {
	compl := []map[string]string{}
	attrs := api.SIMapGetByKeyAsList(htmlConfig, "attrs")
	for _, attrIf := range attrs {
		attrData := api.IAsSIMap(attrIf)
		if api.SIMapGetByKeyIsString(attrData, "scope") {
			if api.SIMapGetByKeyAsString(attrData, "scope") != "all" {
				continue
			}
		} else {
			match := false
			attrScope := api.SIMapGetByKeyAsList(attrData, "scope")
			for _, scopeIf := range attrScope {
				scopeTag := api.IAsString(scopeIf)
				if tagName == scopeTag {
					match = true
				}
			}
			if !match {
				continue
			}
		}
		attrName := api.SIMapGetByKeyAsString(attrData, "name")
		attrDescr := api.SIMapGetByKeyAsString(attrData, "descr")
		attrType := api.SIMapGetByKeyAsString(attrData, "type")
		attrCaption := attrName
		attrValue := attrName
		if attrType != "boolean" {
			attrValue += `=""`
		}
		compl = append(compl, newCompletion(attrValue, attrCaption, attrDescr))
	}
	return compl
}

func effectiveLoopVars(code string, wrap bool) []map[string]string {
	compl := []map[string]string{}
	depth := 0
	for code != "" {
		logger.Debug("vars code:", code)
		rangeIdx := strings.LastIndex(code, "{{range ")
		if rangeIdx == -1 {
			break
		}
		endIdx := strings.LastIndex(code, "{{end}}")
		ifIdx := strings.LastIndex(code, "{{if ")
		if rangeIdx > endIdx {
			if depth <= 0 {
				assignmentIdx := strings.Index(code[rangeIdx:], ":=")
				logger.Debug("assignmentIdx:", assignmentIdx)
				if assignmentIdx != -1 {
					loopVars := strings.Trim(strings.TrimPrefix(code[rangeIdx:rangeIdx+assignmentIdx], "{{range "), " ")
					logger.Debug("loopVars:", loopVars)
					varTokens := strings.Split(loopVars, ",")
					for _, varToken := range varTokens {
						varName := varToken
						if wrap {
							varName = "{" + varName + "}}"
						}
						compl = append(compl, newCompletion(varName, varToken, "loop variable"))
					}
				}
			}
			depth--
			code = code[:rangeIdx]
			continue
		}
		if endIdx > ifIdx {
			if ifIdx == -1 {
				depth++
				code = code[:endIdx]
				continue
			}
			code = code[:ifIdx]
			continue
		} else {
			if ifIdx > -1 {
				code = code[:ifIdx]
				continue
			}
		}
	}
	return compl
}
