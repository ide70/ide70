package codecomplete

import (
	//"fmt"
	"github.com/ide70/ide70/comp"
	"github.com/ide70/ide70/dataxform"
	"reflect"
	"regexp"
	"strings"
)

var reIdentifierStart = regexp.MustCompile(`[+-/*.{;: ()]*\w*$`)

func jsCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	code := yamlPos.valuePrefx
	if code == "" || strings.HasSuffix(code, "(") || strings.HasSuffix(code, ",") || strings.HasSuffix(code, ", ") {
		nrParam, codeWithoutParams := inspectParamBlock(code)
		logger.Info("codeWithoutParams:", codeWithoutParams, "nrParam:", nrParam)
		if nrParam >= 0 {
			tp, funcName :=
				getBaseTypeAndName(codeWithoutParams)
			logger.Info("func:", funcName, "nrParam:", nrParam)
			compl = append(compl,
				completionsOfFuncParam(tp, funcName, nrParam, yamlPos, edContext, configData)...)

			// kikapni tp alapján a paraméter súgót
			// ha nincs VmBase típussal menni
		} else {
			compl = append(compl, completionsOfType(reflect.TypeOf(&comp.VmBase{}), "", configData)...)
		}
		return compl
	}
	if reIdentifierStart.FindString(code) != "" {
		logger.Info("RE tag start")
		tp, funcNamePrefix := getReturnType(code)
		if tp == nil {
			return compl
		}
		compl = append(compl, completionsOfType(tp, funcNamePrefix, configData)...)
	}
	if strings.HasSuffix(code, ")") {
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

func getFuncNameChain(code string) []string {
	funcNameChain := []string{}

	// trim func name fragment
	nameStart := strings.LastIndexAny(code, "+-/*.{;: \n\t()") + 1
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
				nameStart := strings.LastIndexAny(code[:openBracketPos], "+-/*.{;: \n\t") + 1
				funcName := code[nameStart:openBracketPos]
				funcNameChain = append([]string{funcName}, funcNameChain...)
				logger.Info("func name resolved:", funcName)
				if nameStart > 0 {
					code = code[:nameStart]
					continue
				}
			}
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
		baseType = returnTypeOfFunc(baseType, funcName)
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

func getBaseTypeAndName(code string) (reflect.Type, string) {
	funcNameChain := getFuncNameChain(code)

	baseType := reflect.TypeOf(&comp.VmBase{})
	for idx, funcName := range funcNameChain {
		if idx == len(funcNameChain)-1 {
			return baseType, funcName
			logger.Info("rb1", baseType, funcName)
		}
		baseType = returnTypeOfFunc(baseType, funcName)
		if baseType == nil {
			return nil, ""
			logger.Info("rb2")
		}
	}
	logger.Info("rb3", baseType)
	return baseType, ""
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

func completionsOfType(tp reflect.Type, funcNameFilter string, configData map[string]interface{}) []map[string]string {
	compl := []map[string]string{}
	numMethods := tp.NumMethod()
	functionsData := dataxform.SIMapGetByKeyAsMap(configData, "functions")
	typeBasedFunctionsData := dataxform.SIMapGetByKeyAsMap(functionsData, tp.String())

	for i := 0; i < numMethods; i++ {
		method := tp.Method(i)
		logger.Info("methName:", method.Name)
		methodTp := method.Type
		if funcNameFilter != "" && !strings.HasPrefix(method.Name, funcNameFilter) {
			continue
		}
		functionData := dataxform.SIMapGetByKeyAsMap(typeBasedFunctionsData, method.Name)
		functionDescr := dataxform.SIMapGetByKeyAsString(functionData, "descr")
		functionParams := dataxform.SIMapGetByKeyAsList(functionData, "params")
		signature := method.Name + "("
		sigValue := signature
		for inIdx := 1; inIdx < methodTp.NumIn(); inIdx++ {
			inV := methodTp.In(inIdx)
			if inIdx > 1 {
				signature += ", "
				sigValue += ", "
			}
			if inIdx <= len(functionParams) {
				paramDescriptor := dataxform.AsSIMap(functionParams[inIdx-1])
				paramName := dataxform.SIMapGetByKeyAsString(paramDescriptor, "name")
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
	return compl
}

func completionsOfFuncParam(tp reflect.Type, methodName string, paramNo int, yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}) []map[string]string {
	compl := []map[string]string{}
	logger.Info("completionsOfFuncParam")
	functionsData := dataxform.SIMapGetByKeyAsMap(configData, "functions")
	typeBasedFunctionsData := dataxform.SIMapGetByKeyAsMap(functionsData, tp.String())

	functionData := dataxform.SIMapGetByKeyAsMap(typeBasedFunctionsData, methodName)
	functionParams := dataxform.SIMapGetByKeyAsList(functionData, "params")
	paramDescriptor := dataxform.AsSIMap(functionParams[paramNo])
	logger.Info("paramDescriptor:", paramDescriptor)

	completer, configDataCompleter := lookupCompleter("value", paramDescriptor)

	if completer != nil {
		compl = completer(yamlPos, edContext, configDataCompleter, compl)
	}

	return compl
}
