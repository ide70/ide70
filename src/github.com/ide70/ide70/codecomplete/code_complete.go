package codecomplete

import (
	"bytes"
	"github.com/ide70/ide70/dataxform"
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
	child       *YamlPosition
	parent      *YamlPosition
}

type EditorContext struct {
	content string
	row     int
	col     int
}

type ValueCompleter func(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string

var completers map[string]ValueCompleter

func init() {
	completers = map[string]ValueCompleter{
		"jsCompleter":       jsCompleter,
		"fileNameCompleter": fileNameCompleter,
		"yamlDataCompleter": yamlDataCompleter,
		"yamlPathCompleter": yamlPathCompleter}
}

func (yPos *YamlPosition) getKey() string {
	key := yPos.keyPrefix
	if yPos.child != nil {
		key = key + "." + yPos.child.getKey()
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
		if spaces < curSpaces {
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
					logger.Info("findmulti mlvalue:", multilineValue.String())
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
	}
	if len(tokens) > 0 {
		keyPrefix := tokens[0]
		yamlPos.keyPrefix = keyPrefix
		if strings.HasSuffix(keyPrefix, ":") {
			yamlPos.keyPrefix = strings.TrimSuffix(yamlPos.keyPrefix, ":")
			yamlPos.keyComplete = true
			logger.Info("tokens:", tokens, len(tokens))
			logger.Info("keyRow:", keyRow)
			if len(tokens) > 1 {
				valuePrefix := tokens[1]
				logger.Info("vp:", valuePrefix)
				yamlPos.valuePrefx = valuePrefix
			}
		}
	}

	curSpaces := nrOfBeginningSpaces(line)
	if curSpaces > col {
		curSpaces = col
	}
	for ; row >= 0; row-- {
		line = lines[row]
		spaces := nrOfBeginningSpaces(line)
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

func CodeComplete(content string, row, col int, fileType string) []map[string]string {
	lines := preProcessLines(content)
	compl := []map[string]string{}
	yamlPos := getYamlPosition(lines, row, col, true)
	logger.Info("yP:", yamlPos)
	complDescr := loader.GetTemplatedYaml("codeComplete", "").Def
	compDescrFt := dataxform.SIMapGetByKeyAsMap(complDescr, fileType)
	edContext := &EditorContext{content: content, col: col, row: row}

	if len(compDescrFt) > 0 {
		compl = completerCore(yamlPos, edContext, compDescrFt, compl)
	}

	return compl
}

func completerCore(yamlPos *YamlPosition, edContext *EditorContext, levelMap map[string]interface{}, compl []map[string]string) []map[string]string {
	logger.Info("cc yPk:", yamlPos.getKey())
	references := map[string]map[string]interface{}{}
	for {
		logger.Info("matchkeyPrefix:", yamlPos.keyPrefix)
		matchingKeys, perfectMatch, anyMatch := getMatchingKeys(levelMap, yamlPos.keyPrefix)
		logger.Info("pm:", perfectMatch, "am:", anyMatch, "mks:", matchingKeys)
		if perfectMatch {
			keyData := dataxform.IAsSIMap(levelMap[yamlPos.keyPrefix])
			reference := dataxform.SIMapGetByKeyAsString(keyData, "reference")
			if reference != "" {
				references[reference] = levelMap
			}
			if yamlPos.child != nil {
				children := dataxform.SIMapGetByKeyAsMap(keyData, "children")
				if len(children) > 0 {
					levelMap = children
				} else {
					childrenRef := dataxform.SIMapGetByKeyAsString(keyData, "childrenRef")
					if childrenRef != "" {
						levelMap = references[childrenRef]
					} else {
						logger.Info("-- no children")
					}
				}
				yamlPos = yamlPos.child
				continue
			}

			completer, configData := lookupCompleter("value", keyData)
			if completer != nil {
				compl = completer(yamlPos, edContext, configData, compl)
				break
			}
		}
		if anyMatch {
			keyData := dataxform.SIMapGetByKeyAsMap(levelMap, "any")
			logger.Info("kD:", keyData)

			completer, configData := lookupCompleter("key", keyData)

			/*if yamlPos.child != nil {
				children := dataxform.SIMapGetByKeyAsMap(keyData, "children")
				if len(children) > 0 {
					levelMap = children
				} else {
					childrenRef := dataxform.SIMapGetByKeyAsString(keyData, "childrenRef")
					if childrenRef != "" {
						levelMap = references[childrenRef]
					} else {
						logger.Info("-- no children")
					}
				}
				yamlPos = yamlPos.child
				continue
			}

			if completer != nil {
				compl = completer(yamlPos, edContext, configData, compl)
			}*/

			// context switching completer, no children evaluation
			if completer != nil && dataxform.SIMapGetByKeyAsBoolean(configData, "handleChildren") {
				compl = completer(yamlPos, edContext, configData, compl)
			}

			if yamlPos.child != nil {
				children := dataxform.SIMapGetByKeyAsMap(keyData, "children")
				if len(children) > 0 {
					levelMap = children
					yamlPos = yamlPos.child
					continue
				} else {
					childrenRef := dataxform.SIMapGetByKeyAsString(keyData, "childrenRef")
					if childrenRef == "self" {
						logger.Info("childrenRef self")
						yamlPos = yamlPos.child
						continue
					} else if childrenRef != "" {
						logger.Info("childrenRef:", childrenRef)
						levelMap = references[childrenRef]
						yamlPos = yamlPos.child
						continue
					} else {
						logger.Info("-- no children")
						break
					}
				}
			}

			if completer != nil {
				compl = completer(yamlPos, edContext, configData, compl)
			}

			/*if completer != nil {

				if yamlPos.child != nil {
					if !dataxform.SIMapGetByKeyAsBoolean(configData, "handleChildren") {
						logger.Info("completer not applicable")
						break
					}
					children := dataxform.SIMapGetByKeyAsMap(keyData, "children")
					if len(children) > 0 {
						levelMap = children
						yamlPos = yamlPos.child
						continue
					} else {
						childrenRef := dataxform.SIMapGetByKeyAsString(keyData, "childrenRef")
						if childrenRef == "self" {
							logger.Info("childrenRef self")
							yamlPos = yamlPos.child
							continue
						} else if childrenRef != "" {
							levelMap = references[childrenRef]
							yamlPos = yamlPos.child
							continue
						}
					}
				}

				compl = completer(yamlPos, edContext, configData, compl)
			} else {
				if yamlPos.child != nil {
					children := dataxform.SIMapGetByKeyAsMap(keyData, "children")
					if len(children) > 0 {
						levelMap = children
					} else {
						childrenRef := dataxform.SIMapGetByKeyAsString(keyData, "childrenRef")
						if childrenRef == "self" {
							logger.Info("childrenRef self")
						} else if childrenRef != "" {
							levelMap = references[childrenRef]
						} else {
							logger.Info("-- no children")
						}
					}
					yamlPos = yamlPos.child
					continue
				}
			}*/
		}
		logger.Info("pmatch check mks:", matchingKeys)

		for _, matchingKey := range matchingKeys {
			keyPrefix := ""
			keyPostfix := ": "
			logger.Info("matchKey:", matchingKey)
			// short form: key has only a description
			if dataxform.SIMapGetByKeyIsString(levelMap, matchingKey) {
				keyDescr := dataxform.SIMapGetByKeyAsString(levelMap, matchingKey)
				compl = append(compl, newCompletion(keyPrefix+matchingKey+keyPostfix, matchingKey, keyDescr))
				continue
			}
			// complex form: key has spearate descr and other complementary fileds
			keyData := dataxform.SIMapGetByKeyAsMap(levelMap, matchingKey)
			/*keyDescr := dataxform.SIMapGetByKeyAsString(keyData, "descr")
			isListHead := dataxform.SIMapGetByKeyAsBoolean(keyData, "listHead")
			isMapHead := dataxform.SIMapGetByKeyAsBoolean(keyData, "mapHead")
			if isListHead {
				keyPrefix += "- "
			}
			if isMapHead {
				keyPostfix = ":\n" + strings.Repeat(" ", edContext.col+2)
			}
			compl = append(compl, newCompletion(keyPrefix+matchingKey+keyPostfix, matchingKey, keyDescr))*/
			compl = addCompletion(matchingKey, edContext, keyData, compl)
		}

		break
	}
	return compl
}

func addCompletion(value string, edContext *EditorContext, keyData map[string]interface{}, compl []map[string]string) []map[string]string {
	logger.Info("addCompletion:", value)
	keyPrefix := ""
	keyPostfix := ": "
	captionPostfix := ""
	keyDescr := dataxform.SIMapGetByKeyAsString(keyData, "descr")
	isListHead := dataxform.SIMapGetByKeyAsBoolean(keyData, "listHead")
	isMapHead := dataxform.SIMapGetByKeyAsBoolean(keyData, "mapHead")
	isSingleKey := dataxform.SIMapGetByKeyAsBoolean(keyData, "singleKey")
	isValue := dataxform.SIMapGetByKeyAsBoolean(keyData, "value")
	isMultilineValue := dataxform.SIMapGetByKeyAsBoolean(keyData, "multilineValue")
	singleToMap := dataxform.SIMapGetByKeyAsBoolean(keyData, "singleToMap")
	quote := dataxform.SIMapGetByKeyAsString(keyData, "quote")

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
	if quote != "" {
		keyPrefix = quote
		keyPostfix = quote
	}
	logger.Info("finish:")
	return append(compl, newCompletion(keyPrefix+value+keyPostfix, value+captionPostfix, keyDescr))
}

func lookupCompleter(completerType string, keyData map[string]interface{}) (ValueCompleter, map[string]interface{}) {
	completerName := dataxform.SIMapGetByKeyAsString(keyData, completerType+"Completer")
	completerConfig := dataxform.SIMapGetByKeyAsString(keyData, completerType+"CompleterConfig")
	completerParams := dataxform.SIMapGetByKeyAsMap(keyData, completerType+"CompleterParams")
	var configData map[string]interface{} = nil
	if completerConfig != "" {
		configData = loader.GetTemplatedYaml(completerConfig, "").Def
	} else {
		configData = completerParams
	}

	if completerType == "key" {
		dataxform.SIMapCopyKeys(keyData, configData, []string{"descr", "listHead", "mapHead", "singleKey", "multilineValue", "quote"})
	} else {
		configData["value"] = true
	}
	configData["handleChildren"] = completerName == "yamlDataCompleter"

	if completerName != "" {
		logger.Info(completerType+"Completer:", completerName)
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
		logger.Info("lvl key prefix:", k)
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
