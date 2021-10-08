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
		compl = append(compl, completionsOfType(reflect.TypeOf(&comp.VmBase{}), "", configData)...)
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

func getReturnType(code string) (reflect.Type, string) {
	funcNameChain := []string{}

	// trim func name fragment
	nameStart := strings.LastIndexAny(code, "+-/*.{;: \n\t()") + 1
	if nameStart < len(code) {
		funcName := code[nameStart:]
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
	logger.Info("rb3")
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
		methodTp := method.Type
		if funcNameFilter != "" && !strings.HasPrefix(method.Name, funcNameFilter) {
			continue
		}
		functionData := dataxform.SIMapGetByKeyAsMap(typeBasedFunctionsData, method.Name)
		functionDescr := dataxform.SIMapGetByKeyAsString(functionData, "descr")
		functionParams := dataxform.SIMapGetByKeyAsList(functionData, "params")
		signature := method.Name + "("
		for inIdx := 1; inIdx < methodTp.NumIn(); inIdx++ {
			inV := methodTp.In(inIdx)
			if inIdx > 1 {
				signature += ", "
			}
			if inIdx <= len(functionParams) {
				paramName := dataxform.IAsString(functionParams[inIdx-1])
				signature += paramName + ": "
			}
			signature += inV.Name()
		}
		signature += ")"
		sigValue := signature
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
