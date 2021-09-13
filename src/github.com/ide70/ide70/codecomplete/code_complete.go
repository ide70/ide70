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
	child       *YamlPosition
	parent      *YamlPosition
}

type ValueCompleter func(yamlPos *YamlPosition, col int, configData map[string]interface{}, compl []map[string]string) []map[string]string

var completers map[string]ValueCompleter

func init() {
	completers = map[string]ValueCompleter{
		"jsCompleter":       jsCompleter,
		"fileNameCompleter": fileNameCompleter,
		"yamlDataCompleter": yamlDataCompleter}
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

func CodeComplete(content string, row, col int, fileType string) []map[string]string {
	lines := strings.Split(content, "\n")
	for i, _ := range lines {
		// remove cr characters
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}
	compl := []map[string]string{}
	yamlPos := getYamlPosition(lines, row, col, true)
	logger.Info("yP:", yamlPos)
	//compl = append(compl, newCompletion("---\n", "---", "Start yaml document"))
	complDescr := loader.GetTemplatedYaml("codeComplete", "").Def
	compDescrFt := dataxform.SIMapGetByKeyAsMap(complDescr, fileType)

	if len(compDescrFt) > 0 {
		compl = completerCore(yamlPos, col, compDescrFt, compl)
	}

	return compl
}

func completerCore(yamlPos *YamlPosition, col int, levelMap map[string]interface{}, compl []map[string]string) []map[string]string {
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
					}
				}
				yamlPos = yamlPos.child
				continue
			}

			completer, configData := lookupCompleter("value", keyData)
			if completer != nil {
				compl = completer(yamlPos, col, configData, compl)
				break
			}
		}
		if anyMatch {
			keyData := dataxform.SIMapGetByKeyAsMap(levelMap, "any")
			logger.Info("kD:", keyData)

			completer, configData := lookupCompleter("key", keyData)
			if completer != nil {
				compl = completer(yamlPos, col, configData, compl)
			} else {
				if yamlPos.child != nil {
					children := dataxform.SIMapGetByKeyAsMap(keyData, "children")
					if len(children) > 0 {
						levelMap = children
					} else {
						childrenRef := dataxform.SIMapGetByKeyAsString(keyData, "childrenRef")
						if childrenRef != "" {
							levelMap = references[childrenRef]
						}
					}
					yamlPos = yamlPos.child
					continue
				}
			}
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
			keyDescr := dataxform.SIMapGetByKeyAsString(keyData, "descr")
			isListHead := dataxform.SIMapGetByKeyAsBoolean(keyData, "listHead")
			isMapHead := dataxform.SIMapGetByKeyAsBoolean(keyData, "mapHead")
			if isListHead {
				keyPrefix += "- "
			}
			if isMapHead {
				keyPostfix = ":\n" + strings.Repeat(" ", col+2)
			}
			compl = append(compl, newCompletion(keyPrefix+matchingKey+keyPostfix, matchingKey, keyDescr))
		}

		break
	}
	return compl
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
