package codecomplete

import (
	//"fmt"
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
	"regexp"
	"strings"
)

var reWord = regexp.MustCompile(`\w+`)

func htmlCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	code := yamlPos.valuePrefx
	logger.Info("code:", code+"|")

	if strings.HasSuffix(code, "{") {
		compl = append(compl, effectiveLoopVars(code, true)...)

		selfAsTemplatedYaml := loader.ConvertTemplatedYaml([]byte(edContext.content), "self")
		selfData := selfAsTemplatedYaml.Def
		unitInterfaceData := dataxform.SIMapGetByKeyAsMap(selfData, "unitInterface")
		ppData := dataxform.SIMapGetByKeyAsMap(selfData, "privateProperties")
		propertyData := dataxform.SIMapGetByKeyAsMap(unitInterfaceData, "properties")
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
		methods := dataxform.SIMapGetByKeyAsMap(templateConfig, "methods")
		for methodName, methodDataIf := range methods {
			methodData := dataxform.IAsSIMap(methodDataIf)
			closeTag := dataxform.SIMapGetByKeyAsString(methodData, "closeTag")
			methodDescr := dataxform.SIMapGetByKeyAsString(methodData, "descr")
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
		methods := dataxform.SIMapGetByKeyAsMap(templateConfig, "methods")
		logger.Info("meth listed")
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
		logger.Info("meth:", methodName)
		logger.Info("paramNo:", paramNo)

		methodData := dataxform.SIMapGetByKeyAsMap(methods, methodName)

		methodParams := dataxform.SIMapGetByKeyAsList(methodData, "params")
		if paramNo >= len(methodParams) {
			return compl
		}

		compl = append(compl, effectiveLoopVars(code, false)...)

		methodParam := methodParams[paramNo]
		methodParamData := dataxform.IAsSIMap(methodParam)
		logger.Info("meth param data:", methodParamData)
		fixedValue := dataxform.SIMapGetByKeyAsString(methodParamData, "fixedValue")
		paramDescr := dataxform.SIMapGetByKeyAsString(methodParamData, "descr")
		
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
			attrs := dataxform.SIMapGetByKeyAsList(htmlConfig, "attrs")
			for _, attrIf := range attrs {
				attrData := dataxform.IAsSIMap(attrIf)
				attrName := dataxform.SIMapGetByKeyAsString(attrData, "name")
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

	tags := dataxform.SIMapGetByKeyAsList(htmlConfig, "tags")
	for _, tagIf := range tags {
		tagData := dataxform.IAsSIMap(tagIf)
		tagName := dataxform.SIMapGetByKeyAsString(tagData, "name")
		tagDescr := dataxform.SIMapGetByKeyAsString(tagData, "descr")
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
			res[name] = dataxform.SIMapGetByKeyAsString(dt, "descr")
		}
	}
}

func completeAttr(tagName string, htmlConfig map[string]interface{}) []map[string]string {
	compl := []map[string]string{}
	attrs := dataxform.SIMapGetByKeyAsList(htmlConfig, "attrs")
	for _, attrIf := range attrs {
		attrData := dataxform.IAsSIMap(attrIf)
		if dataxform.SIMapGetByKeyIsString(attrData, "scope") {
			if dataxform.SIMapGetByKeyAsString(attrData, "scope") != "all" {
				continue
			}
		} else {
			match := false
			attrScope := dataxform.SIMapGetByKeyAsList(attrData, "scope")
			for _, scopeIf := range attrScope {
				scopeTag := dataxform.IAsString(scopeIf)
				if tagName == scopeTag {
					match = true
				}
			}
			if !match {
				continue
			}
		}
		attrName := dataxform.SIMapGetByKeyAsString(attrData, "name")
		attrDescr := dataxform.SIMapGetByKeyAsString(attrData, "descr")
		attrType := dataxform.SIMapGetByKeyAsString(attrData, "type")
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
		logger.Info("vars code:", code)
		rangeIdx := strings.LastIndex(code, "{{range ")
		if rangeIdx == -1 {
			break
		}
		endIdx := strings.LastIndex(code, "{{end}}")
		ifIdx := strings.LastIndex(code, "{{if ")
		if rangeIdx > endIdx {
			if depth <= 0 {
				assignmentIdx := strings.Index(code[rangeIdx:], ":=")
				logger.Info("assignmentIdx:", assignmentIdx)
				if assignmentIdx != -1 {
					loopVars := strings.Trim(strings.TrimPrefix(code[rangeIdx:rangeIdx+assignmentIdx], "{{range "), " ")
					logger.Info("loopVars:", loopVars)
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
