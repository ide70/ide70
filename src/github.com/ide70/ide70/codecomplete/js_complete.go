package codecomplete

import (
	"fmt"
	"github.com/ide70/ide70/comp"
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/loader"
	"reflect"
	"regexp"
	"strings"
)

var reIdentifierStart = regexp.MustCompile(`[+-/*.{;: ()]*\w*\n*$`)
var reVarNameStart = regexp.MustCompile(`[^.\w]+(\w+)\.\n*$`)
var reWordEnding = regexp.MustCompile(`(\w+)\n*$`)
var reVarDefConsts = regexp.MustCompile(`var \w+[^(]+\("(\w+)"\)`)
var reVarFirstFuncName = regexp.MustCompile(`var \w+\s*=\s*(\.|\w+[^(\w]|\w+\(\))*(\w+)\("`)
var reVariableDefinition = regexp.MustCompile(`var (\w+)\s*=\s*([^;]+)`)
var reVarParentVar = regexp.MustCompile(`var \w+\s*=\s*(\w+)\.|var \w+\s*=\s*(\w+);`)

func jsCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	edContext.contextType = "js"
	code := yamlPos.valuePrefx
	logger.Info("code:", code+"|")
	if code == "" || strings.HasSuffix(code, "(") || strings.HasSuffix(code, ",") || strings.HasSuffix(code, ", ") {
		nrParam, codeWithoutParams := inspectParamBlock(code)
		logger.Info("codeWithoutParams:", codeWithoutParams, "nrParam:", nrParam)
		if nrParam >= 0 {
			tp, funcName, attrs :=
				getBaseTypeAndName(codeWithoutParams)
			logger.Info("func:", funcName, "nrParam:", nrParam)
			
			compls := completionsOfFuncParam(tp, funcName, nrParam, yamlPos, edContext, configData, attrs)
			if len(compls) > 0 {
				logger.Info("compls found:", len(compls))
				compl = append(compl, compls...)
			} else {
				logger.Info("no compls found")
				compl = append(compl, completionsOfType(reflect.TypeOf(&comp.VmBase{}), "", yamlPos, edContext, configData)...)
				compl = append(compl, collectVarDefs(code)...)
			}
		} else {
			compl = append(compl, completionsOfType(reflect.TypeOf(&comp.VmBase{}), "", yamlPos, edContext, configData)...)
			compl = append(compl, collectVarDefs(code)...)
		}
		return compl
	}
	varNameStartMatches := reVarNameStart.FindStringSubmatch(code)
	if varNameStartMatches != nil {
		varNameStart := varNameStartMatches[1]
		logger.Info("varNameStart:", varNameStart)

		tp, varAttributes := getVarType(code, varNameStart)

		if tp == nil {
			return compl
		}
		configDataForCompl := api.SIMapLightCopy(configData)
		configDataForCompl["firstConst"] = varAttributes["firstConst"]
		configDataForCompl["table"] = varAttributes["table"]
		logger.Info("firstConst:", varAttributes["firstConst"])
		compl = append(compl, completionsOfType(tp, "", yamlPos, edContext, configDataForCompl)...)
		return compl
	}
	if reIdentifierStart.FindString(code) != "" {
		logger.Info("RE tag start:")
		tp, funcNamePrefix := getReturnType(code)
		if tp == nil {
			return compl
		}
		logger.Info("return type: ", tp.String())
		logger.Info("funcNamePrefix: ", funcNamePrefix)
		compl = append(compl, completionsOfType(tp, funcNamePrefix, yamlPos, edContext, configData)...)
		if tp == reflect.TypeOf(&comp.VmBase{}) {
			compl = append(compl, collectVarDefs(code)...)
		}
	}
	if strings.HasSuffix(code, ")") {
	}
	return compl
}

func getVarType(code, varName string) (reflect.Type, map[string]string) {
	defs := availableVarDefs(code)
	varDef := defs[varName]
	logger.Info("vdef:", varDef)
	identifierDef := varDef.defCode
	logger.Info("identifierDef:", identifierDef)
	// local variable and its definition found
	if identifierDef != "" {
		def := identifierDef + "."
		tp, _ := getReturnType(def)
		varAttributes := calcVarAttributes(varName, defs)
		varAttributes["firstConst"] = varDef.firstConst
		return tp, varAttributes
	}
	return nil, nil
}

func yamlLookup(folderPrefix, fileName string, yamlExpr string) string {
	fileAsTemplatedYaml := loader.GetTemplatedYaml(fileName, "ide70/"+folderPrefix+"/")
	reExpr, isFilterValue := convertYamlpathToRegex(yamlExpr, nil)
	logger.Info("filterExpr:", reExpr)
	value := ""
	if reExpr != nil {
		logger.Info("examine nodes")
		api.IApplyFnToNodes(fileAsTemplatedYaml.IDef, func(entry api.CollectionEntry) {
			logger.Info("lin key:", entry.LinearKey())
			if reExpr.MatchString(entry.LinearKey()) {
				value, _ = leafValDescr(entry, isFilterValue)
				return
			}
		})
		logger.Info("examine nodes done")
	}
	return value
}

func calcVarAttributes(varName string, defs map[string]VarDef) map[string]string {
	attributes := map[string]string{}
	// table
	attributeName := "table"
	def := defs[varName]
	if def.firstFuncName == "Table" {
		attributes[attributeName] = def.firstConst
	}
	if def.firstFuncName == "JoinedTable" {
		parentAttributes := calcVarAttributes(def.parentVarName, defs)
		attributes[attributeName] = yamlLookup("dcfg/schema", parentAttributes[attributeName], fmt.Sprintf("connections.%s.foreignTable:value", def.firstConst))
	}
	logger.Info("attrs for var ", varName, attributes)
	return attributes
}

func filterNonvisibleScope(code string) string {
	res := ""
	depth := 0
	for {
		braceIdx := strings.LastIndexAny(code, "{}")
		if braceIdx == -1 {
			if depth <= 0 {
				res = code + res
				return res
			}
		} else {
			if depth <= 0 {
				res = code[braceIdx:] + res
			} else {
				res = string(code[braceIdx]) + res
			}
			brace := code[braceIdx]
			if brace == '{' {
				depth--
			} else {
				depth++
			}
		}
		code = code[:braceIdx]
	}
}

type VarDef struct {
	defCode       string
	firstConst    string
	firstFuncName string
	parentVarName string
}

func findLastStringSubmatch(re *regexp.Regexp, s string) []string {
	matches := re.FindAllStringSubmatch(s, -1)
	if matches != nil {
		return matches[len(matches)-1]
	}
	return nil
}

func availableVarDefs(code string) map[string]VarDef {
	defs := map[string]VarDef{}
	code = filterNonvisibleScope(code)
	varDefs := reVariableDefinition.FindAllStringSubmatch(code, -1)
	varDefPositions := reVariableDefinition.FindAllStringSubmatchIndex(code, -1)
	for idx, varDefMatch := range varDefs {
		//logger.Info("vardef:", varDefMatch[1], code[:varDefPositions[idx][1]])
		//defs[varDefMatch[1]] = varDefMatch[2]
		defCode := code[:varDefPositions[idx][1]]
		logger.Info("defCode:",defCode)
		varDef := VarDef{defCode: defCode}

		constMatch := findLastStringSubmatch(reVarDefConsts, defCode)
		logger.Info("constMatch:",constMatch)
		if constMatch != nil {
			varDef.firstConst = constMatch[1]
		}

		parentVarMatch := findLastStringSubmatch(reVarParentVar, defCode)
		logger.Info("parentVarMatch:",parentVarMatch)
		if parentVarMatch != nil {
			varDef.parentVarName = parentVarMatch[1]
		}

		constFuncNameMatch := findLastStringSubmatch(reVarFirstFuncName, defCode)
		logger.Info("constFuncNameMatch:",constFuncNameMatch)
		if constFuncNameMatch != nil {
			varDef.firstFuncName = constFuncNameMatch[2]
		}

		defs[varDefMatch[1]] = varDef
		logger.Info("vardef:", varDefMatch[1], varDef)
	}
	return defs
}

func collectVarDefs(code string) []map[string]string {
	compl := []map[string]string{}
	code = filterNonvisibleScope(code)
	logger.Info("vScope:", code+"|")
	varDefs := reVariableDefinition.FindAllStringSubmatch(code, -1)
	for _, varDefMatch := range varDefs {
		compl = append(compl, newCompletion(varDefMatch[1], varDefMatch[1], "local variable"))
	}
	return compl
}

func inspectParamBlock(code string) (int, string) {
	paramNo := 0
	nrClosingBracket := 0
	for len(code) > 0 {
		if strings.HasSuffix(code, ")") {
			nrClosingBracket++
		}
		if strings.HasSuffix(code, "(") {
			if nrClosingBracket == 0 {
				return paramNo, strings.TrimSuffix(code, "(")
			} else {
				nrClosingBracket--
			}
		}
		if nrClosingBracket == 0 && strings.HasSuffix(code, ",") {
			paramNo++
		}
		code = code[:len(code)-1]
	}
	return -1, code
}

func removeLeftWhiteSpace(code string) string {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimLeft(line, " \t")
	}
	return strings.Join(lines, "")
	/*currLineStartPos := strings.LastIndex(code, "\n") + 1
	if currLineStartPos == -1 {
		return code
	}
	prevLine := code[:currLineStartPos-1]
	currLine := code[currLineStartPos:]
	currLine = strings.TrimLeft(currLine, " \t")
	return prevLine + currLine*/
}

func getFuncNameChain(code string) []string {
	funcNameChain := []string{}

	code = removeLeftWhiteSpace(code)
	// trim func name fragment
	nameStart := strings.LastIndexAny(code, "+-/*.{}=;: \n\t()") + 1
	if nameStart < len(code) {
		funcName := code[nameStart:]
		logger.Info("getReturnType funcName:", funcName)
		funcNameChain = append([]string{funcName}, funcNameChain...)
		code = code[:nameStart]
	}

	for strings.HasSuffix(code, ".") {
		code = strings.TrimSuffix(code, ".")

		if strings.HasSuffix(code, ")") {
			openBracketPos := findOpeningBracket(code, len(code)-1)
			if openBracketPos != -1 {
				nameStart := strings.LastIndexAny(code[:openBracketPos], "+-/*.{}=;,(: \n\t") + 1
				funcName := code[nameStart:openBracketPos]
				funcNameChain = append([]string{funcName}, funcNameChain...)
				logger.Info("func name resolved:", funcName)
				if nameStart > 0 {
					code = code[:nameStart]
					continue
				}
			}
		} else if match := reWordEnding.FindStringSubmatch(code); match != nil {
			logger.Info("var name resolved:", match[1])
			funcNameChain = append([]string{"var:" + match[1]}, funcNameChain...)
			code = strings.TrimSuffix(code, match[0])
			continue
		}
		break
	}

	return funcNameChain
}

func getReturnType(code string) (reflect.Type, string) {
	funcNameChain := getFuncNameChain(code)

	baseType := reflect.TypeOf(&comp.VmBase{})
	for idx, funcName := range funcNameChain {
		baseTypePrev := baseType
		if strings.HasPrefix(funcName, "var:") {
			if baseType.Kind() == reflect.Map {
				logger.Info("basetype map type, tread var as map element")
				baseType = baseType.Elem()
			} else {
				varName := strings.TrimPrefix(funcName, "var:")
				baseType, _ = getVarType(code, varName)
			}
		} else {
			baseType = returnTypeOfFunc(baseType, funcName)
		}
		if baseType == nil {
			if idx == len(funcNameChain)-1 {
				return baseTypePrev, funcName
				logger.Info("rb1", baseType, funcName)
			}
			return nil, ""
			logger.Info("rb2")
		}
	}
	logger.Info("rb3", baseType)
	return baseType, ""
}

func getBaseTypeAndName(code string) (reflect.Type, string, map[string]string) {
	funcNameChain := getFuncNameChain(code)
	attrs := map[string]string{}

	baseType := reflect.TypeOf(&comp.VmBase{})
	for idx, funcName := range funcNameChain {
		if idx == len(funcNameChain)-1 {
			return baseType, funcName, attrs
			logger.Info("rb1", baseType, funcName)
		}
		if strings.HasPrefix(funcName, "var:") {
			varName := strings.TrimPrefix(funcName, "var:")
			baseType, attrs = getVarType(code, varName)
		} else {
			baseType = returnTypeOfFunc(baseType, funcName)
		}
		if baseType == nil {
			return nil, "", nil
			logger.Info("rb2")
		}
	}
	logger.Info("rb3", baseType)
	return baseType, "", attrs
}

func returnTypeOfFunc(baseType reflect.Type, funcName string) reflect.Type {
	logger.Info("returnTypeOfFunc", baseType, funcName)
	method, has := baseType.MethodByName(funcName)
	if !has {
		return nil
	}
	methodTp := method.Type
	if methodTp.NumOut() == 1 {
		return methodTp.Out(0)
	}
	return nil
}

func findOpeningBracket(code string, pos int) int {
	nrClose := 1
	for i := pos - 1; i >= 0; i-- {
		if code[i] == '(' {
			nrClose--
		}
		if nrClose == 0 {
			return i
		}
		if code[i] == ')' {
			nrClose++
		}
	}
	return -1
}

func completionsOfType(tp reflect.Type, funcNameFilter string, yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}) []map[string]string {
	compl := []map[string]string{}
	numMethods := tp.NumMethod()
	functionsData := api.SIMapGetByKeyAsMap(configData, "functions")
	typeBasedFunctionsData := api.SIMapGetByKeyAsMap(functionsData, tp.String())
	typesData := api.SIMapGetByKeyAsMap(configData, "types")
	typeBasedTypeData := api.SIMapGetByKeyAsMap(typesData, tp.String())

	for i := 0; i < numMethods; i++ {
		method := tp.Method(i)
		logger.Info("methName:", method.Name)
		methodTp := method.Type
		if funcNameFilter != "" && !strings.HasPrefix(method.Name, funcNameFilter) {
			continue
		}
		functionData := api.SIMapGetByKeyAsMap(typeBasedFunctionsData, method.Name)
		functionDescr := api.SIMapGetByKeyAsString(functionData, "descr")
		functionParams := api.SIMapGetByKeyAsList(functionData, "params")
		signature := method.Name + "("
		sigValue := signature
		for inIdx := 1; inIdx < methodTp.NumIn(); inIdx++ {
			inV := methodTp.In(inIdx)
			if inIdx > 1 {
				signature += ", "
				sigValue += ", "
			}
			if inIdx <= len(functionParams) {
				paramDescriptor := api.AsSIMap(functionParams[inIdx-1])
				paramName := api.SIMapGetByKeyAsString(paramDescriptor, "name")
				signature += paramName + ": "
			}
			signature += inV.Name()
		}
		signature += ")"
		sigValue += ")"
		for outIdx := 0; outIdx < methodTp.NumOut(); outIdx++ {
			outV := methodTp.Out(outIdx)
			if outIdx > 0 {
				signature += ","
			}
			outVName := " " + outV.String()
			signature += outVName
		}

		compl = append(compl, newCompletion(sigValue, signature, functionDescr))
	}

	fieldCompleter := api.SIMapGetByKeyAsMap(typeBasedTypeData, "fieldCompleter")
	if len(fieldCompleter) == 1 {
		logger.Info("fieldCompleter found for type:", tp.String())
		completer, configDataCompleter := lookupCompleter("value", fieldCompleter)

		if completer != nil {
			configDataCompleter["firstConst"] = configData["firstConst"]
			configDataCompleter["table"] = configData["table"]
			logger.Info("configDataCompleter:", configDataCompleter)
			compl = completer(yamlPos, edContext, configDataCompleter, compl)
		}
	}

	return compl
}

func completionsOfFuncParam(tp reflect.Type, methodName string, paramNo int, yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, attrs map[string]string) []map[string]string {
	compl := []map[string]string{}
	if tp == nil {
		return compl
	}
	logger.Info("completionsOfFuncParam, base type:", tp.String())
	functionsData := api.SIMapGetByKeyAsMap(configData, "functions")

	typeBasedFunctionsData := api.SIMapGetByKeyAsMap(functionsData, tp.String())
	functionData := api.SIMapGetByKeyAsMap(typeBasedFunctionsData, methodName)
	functionParams := api.SIMapGetByKeyAsList(functionData, "params")
	if paramNo >= len(functionParams) {
		return compl
	}

	paramDescriptor := api.AsSIMap(functionParams[paramNo])
	logger.Info("paramDescriptor:", paramDescriptor)

	completer, configDataCompleter := lookupCompleter("value", paramDescriptor)

	if completer != nil {
		configDataCompleter["firstConst"] = attrs["firstConst"]
		configDataCompleter["table"] = attrs["table"]
		compl = completer(yamlPos, edContext, configDataCompleter, compl)
	}

	return compl
}
