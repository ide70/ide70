package codecomplete

import (
	"bytes"
	"fmt"
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/loader"
	"github.com/ide70/ide70/util/log"
	"strings"
)

/*

concepts:
yaml locations:
0,0 empty document
0,col: root level keys

*/

var logger = log.Logger{"codecomplete"}

type YamlPosition struct {
	keyPrefix   string
	valuePrefx  string
	keyComplete bool
	inNextLine  bool
	isArray     bool
	arrayPos    int
	child       *YamlPosition
	parent      *YamlPosition
}

type EditorContext struct {
	content     string
	row         int
	col         int
	keyStartCol int
	contextType string
}

type ValueCompleter func(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string

var completers map[string]ValueCompleter

func init() {
	completers = map[string]ValueCompleter{
		"jsCompleter":          jsCompleter,
		"fileNameCompleter":    fileNameCompleter,
		"fileContentCompleter": fileContentCompleter,
		"yamlDataCompleter":    yamlDataCompleter,
		"yamlPathCompleter":    yamlPathCompleter,
		"idCompleter":          idCompleter,
		"htmlCompleter":        htmlCompleter,
		"dictCompleter":        dictCompleter,
		"templateCompleter":    templateCompleter,
		"union":                unionCompleter,
		"firstOf":              unionCompleter}
}

func (yPos *YamlPosition) getKey() string {
	key := yPos.keyPrefix
	if yPos.child != nil {
		key = key + "." + yPos.child.getKey()
	}
	return key
}

func (yPos *YamlPosition) getIndexedKey() string {
	key := ""
	if yPos.parent != nil {
		key = yPos.parent.getIndexedKey()
	}
	if yPos.isArray {
		key += fmt.Sprintf("[%d]", yPos.arrayPos)
	}
	if !yPos.isArray || yPos.child == nil || yPos.child.isArray {
		if key != "" {
			key += "."
		}
		key += yPos.keyPrefix
	}
	return key
}

func nrOfBeginningSpaces(line string) int {
	count := 0
	for _, c := range line {
		if c == ' ' {
			count++
		} else {
			return count
		}
	}
	return count
}

func findMultilineValue(lines []string, row, col int) (string, int) {
	curSpaces := nrOfBeginningSpaces(lines[row])
	if curSpaces > col {
		curSpaces = col
	}

	keySpaces := curSpaces
	for srchRow := row; srchRow >= 0; srchRow-- {
		line := lines[srchRow]
		spaces := nrOfBeginningSpaces(line)
		if spaces < keySpaces {
			if strings.HasSuffix(line, ":") {
				keySpaces = nrOfBeginningSpaces(line)
			} else if strings.HasSuffix(line, ": |") {
				if spaces < keySpaces {
					var multilineValue bytes.Buffer
					valSpaces := nrOfBeginningSpaces(lines[srchRow+1])
					for valRow := srchRow + 1; valRow < row; valRow++ {
						multilineValue.WriteString(lines[valRow][valSpaces:] + "\n")
					}
					multilineValue.WriteString(lines[row][valSpaces:col])
					logger.Debug("findmulti mlvalue:", multilineValue.String())
					return multilineValue.String(), srchRow
				}
			}
		}
	}
	return "", -1
}

func getYamlPosition(lines []string, row, col int, findMultiline bool) *YamlPosition {
	yamlPos := &YamlPosition{}
	keyRow := -1
	if findMultiline {
		yamlPos.valuePrefx, keyRow = findMultilineValue(lines, row, col)
	}

	if keyRow != -1 {
		if row == keyRow+1 {
			yamlPos.inNextLine = true
		}
		row = keyRow
		col = strings.Index(lines[row], ":")
	}

	line := lines[row]
	prefix := strings.TrimLeft(line[:col], " ")
	tokens := strings.Split(prefix, " ")

	if len(tokens) > 0 && tokens[0] == "-" {
		tokens = tokens[1:]
		yamlPos.isArray = true
	}
	if len(tokens) > 0 {
		keyPrefix := tokens[0]
		yamlPos.keyPrefix = keyPrefix
		if strings.HasSuffix(keyPrefix, ":") {
			yamlPos.keyPrefix = strings.TrimSuffix(yamlPos.keyPrefix, ":")
			yamlPos.keyComplete = true
			logger.Debug("tokens:", tokens, len(tokens))
			logger.Debug("keyRow:", keyRow)
			if len(tokens) > 1 {
				valuePrefix := tokens[1]
				logger.Debug("vp:", valuePrefix)
				yamlPos.valuePrefx = valuePrefix
			}
		}
	}

	curSpaces := nrOfBeginningSpaces(line)
	if curSpaces > col {
		curSpaces = col
	}
	for row--; row >= 0; row-- {
		line = lines[row]
		spaces := nrOfBeginningSpaces(line)
		if spaces == curSpaces && strings.HasPrefix(line[spaces:], "- ") {
			yamlPos.arrayPos++
		}
		if spaces < curSpaces {
			//idxColon := strings.Index(line, ":")
			yamlPosParent := getYamlPosition(lines, row, len(line), false)
			//yamlPosParent := getYamlPosition(lines, row, idxColon, false)
			yamlPosParentLastChild := yamlPosParent
			for yamlPosParentLastChild.child != nil {
				yamlPosParentLastChild = yamlPosParentLastChild.child
			}
			yamlPosParentLastChild.child = yamlPos
			yamlPos.parent = yamlPosParentLastChild
			yamlPos = yamlPosParent
			break
		}
	}

	return yamlPos
}

func preProcessLines(content string) []string {
	lines := strings.Split(content, "\n")
	for i, _ := range lines {
		// remove cr characters
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}
	return lines
}

func CodeComplete(content string, row, col int, fileName string) []map[string]string {
	lines := preProcessLines(content)
	compl := []map[string]string{}

	complDescr := loader.GetTemplatedYaml("codeComplete", "").Def
	maxPrefix := ""
	for k, _ := range complDescr {
		if strings.HasPrefix(fileName, k) {
			if len(k) > len(maxPrefix) {
				maxPrefix = k
			}
		}
	}

	if maxPrefix == "" {
		logger.Debug("unknown prefix:", fileName)
		return compl
	}

	compDescrFt := api.SIMapGetByKeyAsMap(complDescr, maxPrefix)
	edContext := &EditorContext{content: content, col: col, row: row, keyStartCol: nrOfBeginningSpaces(lines[row])}

	if len(compDescrFt) > 0 {
		yamlPos := getYamlPosition(lines, row, col, true)
		logger.Debug("yP:", yamlPos)
		compl = completerCore(yamlPos, edContext, compDescrFt, compl)
	}

	headerText := ""
	headerIdx := -1
	for idx, complItem := range compl {
		if complItem["header"] != "" {
			headerText = complItem["header"]
			headerIdx = idx
		}
	}

	if headerText != "" {
		sliceDeleteAt(&compl, headerIdx)
		for _, complItem := range compl {
			complItem["docHTML"] = "<div class='complete-doc-hdr'>"+headerText+"</div>"
		}
	}

	return compl
}

func sliceDeleteAt(sp *[]map[string]string, i int) {
	*sp = append((*sp)[:i], (*sp)[i+1:]...)
}

func completerCore(yamlPos *YamlPosition, edContext *EditorContext, levelMap map[string]interface{}, compl []map[string]string) []map[string]string {
	logger.Debug("cc yPk:", yamlPos.getKey())
	references := map[string]map[string]interface{}{}
	for {
		logger.Debug("matchkeyPrefix:", yamlPos.keyPrefix)
		matchingKeys, perfectMatch, anyMatch := getMatchingKeys(levelMap, yamlPos.keyPrefix)
		logger.Debug("pm:", perfectMatch, "am:", anyMatch, "mks:", matchingKeys)
		if perfectMatch {
			keyData := api.IAsSIMap(levelMap[yamlPos.keyPrefix])
			reference := api.SIMapGetByKeyAsString(keyData, "reference")
			if reference != "" {
				references[reference] = levelMap
			}
			if yamlPos.child != nil {
				children := api.SIMapGetByKeyAsMap(keyData, "children")
				if len(children) > 0 {
					levelMap = children
				} else {
					childrenRef := api.SIMapGetByKeyAsString(keyData, "childrenRef")
					if childrenRef != "" {
						levelMap = references[childrenRef]
					} else {
						logger.Debug("-- no children")
					}
				}
				yamlPos = yamlPos.child
				continue
			}

			if yamlPos.keyComplete {
				logger.Debug("-- complete key")
				possibleValues := api.SIMapGetByKeyAsMap(keyData, "possibleValues")
				if len(possibleValues) > 0 {
					matchingKeys = []string{}
					for value, descrIf := range possibleValues {
						compl = addValueCompletion(value, api.IAsString(descrIf), edContext, keyData, compl)
					}
				}
			}

			completer, configData := lookupCompleter("value", keyData)
			if completer != nil {
				if keyData["descr"] != nil {
					compl = append(compl, newHeader(api.SIMapGetByKeyAsString(keyData,"descr")))
				}
				compl = completer(yamlPos, edContext, configData, compl)
				break
			}
		}
		if anyMatch {
			keyData := api.SIMapGetByKeyAsMap(levelMap, "any")
			logger.Debug("kD:", keyData)
			reference := api.SIMapGetByKeyAsString(keyData, "reference")
			if reference != "" {
				references[reference] = levelMap
			}

			completer, configData := lookupCompleter("key", keyData)

			// context switching completer, no children evaluation
			if completer != nil && api.SIMapGetByKeyAsBoolean(configData, "handleChildren") {
				compl = completer(yamlPos, edContext, configData, compl)
			}

			if yamlPos.child == nil && completer == nil {
				preventBlankKey := api.SIMapGetByKeyAsBoolean(keyData, "preventBlankKey")
				if !preventBlankKey {
					compl = addCompletion("___", edContext, keyData, compl)
				}
			}

			if yamlPos.child != nil {
				children := api.SIMapGetByKeyAsMap(keyData, "children")
				if len(children) > 0 {
					levelMap = children
					yamlPos = yamlPos.child
					continue
				} else {
					childrenRef := api.SIMapGetByKeyAsString(keyData, "childrenRef")
					if childrenRef == "self" {
						logger.Debug("childrenRef self")
						yamlPos = yamlPos.child
						continue
					} else if childrenRef != "" {
						logger.Debug("childrenRef:", childrenRef)
						levelMap = references[childrenRef]
						logger.Debug("levelMap:", levelMap)
						yamlPos = yamlPos.child
						continue
					} else {
						logger.Debug("-- no children")
						break
					}
				}
			}

			if completer != nil {
				compl = completer(yamlPos, edContext, configData, compl)
			}

		}
		logger.Debug("pmatch check mks:", matchingKeys)

		for _, matchingKey := range matchingKeys {
			keyPrefix := ""
			keyPostfix := ": "
			logger.Debug("matchKey:", matchingKey)
			// short form: key has only a description
			if api.SIMapGetByKeyIsString(levelMap, matchingKey) {
				keyDescr := api.SIMapGetByKeyAsString(levelMap, matchingKey)
				compl = append(compl, newCompletion(keyPrefix+matchingKey+keyPostfix, matchingKey, keyDescr))
				continue
			}
			// complex form: key has spearate descr and other complementary fileds
			keyData := api.SIMapGetByKeyAsMap(levelMap, matchingKey)
			compl = addCompletion(matchingKey, edContext, keyData, compl)
		}

		break
	}
	return compl
}

func addValueCompletion(value, descr string, edContext *EditorContext, keyData map[string]interface{}, compl []map[string]string) []map[string]string {
	logger.Debug("addValueCompletion:", value)
	keyPrefix := ""
	captionPostfix := ""
	quote := api.SIMapGetByKeyAsString(keyData, "quote")
	descrPrefix := api.SIMapGetByKeyAsString(keyData, "descrPrefix")
	descrPostfix := api.SIMapGetByKeyAsString(keyData, "descrPostfix")
	keyPostfix := ""
	if edContext.contextType != "template" && edContext.contextType != "js" {
		keyPostfix = "\n" + strings.Repeat(" ", edContext.keyStartCol)
	}
	if quote != "" {
		keyPrefix = quote
		keyPostfix = quote + keyPostfix
	}
	if descrPrefix != "" {
		descr = descrPrefix + " - " + descr
	}
	if descrPostfix != "" {
		descr = descr + " (" + descrPostfix + ")"
	}
	return append(compl, newCompletion(keyPrefix+value+keyPostfix, value+captionPostfix, descr))
}

func addCompletion(value string, edContext *EditorContext, keyData map[string]interface{}, compl []map[string]string) []map[string]string {
	logger.Debug("addCompletion:", value)
	keyPrefix := ""
	keyPostfix := ": "
	captionPostfix := ""
	keyDescr := api.SIMapGetByKeyAsString(keyData, "descr")
	isListHead := api.SIMapGetByKeyAsBoolean(keyData, "listHead")
	isMapHead := api.SIMapGetByKeyAsBoolean(keyData, "mapHead")
	isSingleKey := api.SIMapGetByKeyAsBoolean(keyData, "singleKey")
	isValue := api.SIMapGetByKeyAsBoolean(keyData, "value")
	isMultilineValue := api.SIMapGetByKeyAsBoolean(keyData, "multilineValue")
	singleToMap := api.SIMapGetByKeyAsBoolean(keyData, "singleToMap")
	quote := api.SIMapGetByKeyAsString(keyData, "quote")
	subProperties := api.SIMapGetByKeyAsList(keyData, "subProperties")

	if singleToMap {
		captionPostfix = " :"
	}

	if isSingleKey || isValue {
		keyPostfix = ""
	}
	if isListHead {
		keyPrefix += "- "
	}
	if isMapHead {
		newCol := edContext.col + 2
		if singleToMap && len(value) < edContext.col+2 {
			newCol = edContext.col + 2 - len(value)
		}
		keyPostfix = ":\n" + strings.Repeat(" ", newCol)
	}
	if isMultilineValue {
		keyPostfix = ": |\n" + strings.Repeat(" ", edContext.col+2)
	}
	for _, subPropertyIf := range subProperties {
		subProperty := api.IAsString(subPropertyIf)
		if subProperty != "" {
			keyPostfix += "\n" + strings.Repeat(" ", edContext.keyStartCol+2) + subProperty + ": "
		}
	}
	if quote != "" {
		keyPrefix = quote
		keyPostfix = quote
	}
	logger.Debug("finish:")
	return append(compl, newCompletion(keyPrefix+value+keyPostfix, value+captionPostfix, keyDescr))
}

func lookupCompleter(completerType string, keyData map[string]interface{}) (ValueCompleter, map[string]interface{}) {
	completerKey := completerType + "Completer"
	completerMap := api.SIMapGetByKeyAsMap(keyData, completerKey)
	if len(completerMap) != 1 {
		logger.Debug("no completer / multiple completers")
		logger.Debug("completerType:", completerType)
		return nil, nil
	}
	completerName, completerParamsIf := api.GetOnlyEntry(completerMap)

	if completerName == "completerRef" {
		refValue := api.IAsString(completerParamsIf)
		fileAsTemplatedYaml := loader.GetTemplatedYaml("namedCompleters", "ide70/dcfg/")
		namedCompleters := fileAsTemplatedYaml.Def
		completerData := api.SIMapGetByKeyAsMap(namedCompleters, refValue)
		if len(completerData) == 0 {
			return nil, nil
		}
		completerDef := api.SIMapGetByKeyAsMap(completerData, "definition")
		completerName, completerParamsIf = api.GetOnlyEntry(completerDef)
	}

	completerParamsList := api.IAsArr(completerParamsIf)
	completerParams := api.IAsSIMap(completerParamsIf)
	if len(completerParamsList) > 0 {
		completerParams["paramsList"] = completerParamsList
		completerParams["completerType"] = completerType
	}
	configFile := api.SIMapGetByKeyAsString(completerParams, "configFile")
	var configData map[string]interface{} = nil
	if configFile != "" {
		configData = loader.GetTemplatedYaml(configFile, "").Def
	} else {
		configData = completerParams
	}

	if completerType == "key" {
		api.SIMapCopyKeys(keyData, configData, []string{"descr", "listHead", "mapHead", "singleKey", "multilineValue", "quote"})
	} else {
		configData["value"] = true
	}
	
	configData["handleChildren"] = false
	if len(completerParamsList) > 0 {
		logger.Debug("range completerParamsList")
		for _,subCompleterIf := range completerParamsList {
			key,_ := api.GetOnlyEntry(api.IAsSIMap(subCompleterIf))
			logger.Debug("key", key)
			if key == "yamlDataCompleter" {
				configData["handleChildren"] = true
			}
		}
	}
	if completerName == "yamlDataCompleter" {
		configData["handleChildren"] = true
	}
	
	if completerName == "firstOf" {
		configData["firstNonemptyOnly"] = true
	}

	if completerName != "" {
		logger.Debug(completerType+"Completer:", completerName)
		completer := completers[completerName]
		if completer != nil {
			return completer, configData
		}
	}
	return nil, nil
}

func getMatchingKeys(level map[string]interface{}, keyPrefix string) ([]string, bool, bool) {
	matching := []string{}
	perfectMatch := false
	anyMatch := false
	for k, _ := range level {
		logger.Debug("lvl key prefix:", k)
		if k == "any" {
			//matching = append(matching, keyPrefix)
			anyMatch = true
			continue
		}
		if strings.HasPrefix(k, keyPrefix) {
			matching = append(matching, k)
			if k == keyPrefix {
				perfectMatch = true
			}
		}
	}
	return matching, perfectMatch, anyMatch
}

func newCompletion(value, caption, meta string) map[string]string {
	completion := map[string]string{}
	completion["value"] = value
	completion["caption"] = caption
	completion["meta"] = meta
	return completion
}

func newHeader(header string) map[string]string {
	completion := map[string]string{}
	completion["header"] = header
	return completion
}
