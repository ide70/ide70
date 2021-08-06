package server

import (
	"bytes"
	"encoding/json"
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
	"net/http"
	"strconv"
	"strings"
)

/*

concepts:
yaml locations:
0,0 empty document
0,col: root level keys

*/

type YamlPosition struct {
	keyPrefix   string
	valuePrefx  string
	keyComplete bool
	child       *YamlPosition
}

type ValueCompleter func(yamlPos *YamlPosition, compl []map[string]string) []map[string]string

var valueCompleters map[string]ValueCompleter = map[string]ValueCompleter{"jsCompleter": jsCompleter}

func (s *AppServer) serveCodeComplete(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	parts := strings.Split(r.URL.Path, "/")

	if len(parts) < 3 {
		// Missing app name from path
		http.NotFound(w, r)
		return
	}
	// Omit the first empty string, app name and pathStatic
	parts = parts[3:]

	content := r.FormValue("content")
	row, _ := strconv.ParseInt(r.FormValue("row"), 10, 32)
	col, _ := strconv.ParseInt(r.FormValue("col"), 10, 32)
	fileName := strings.Join(parts, "/")
	fileType := parts[1]
	logger.Info("Code complete file name:", fileName, fileType, int(row), int(col))

	completions := codeComplete(content, int(row), int(col), fileType)
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(completions)
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
			if keyRow != -1 && len(tokens) > 1 {
				valuePrefix := tokens[1]
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
			idxColon := strings.Index(line, ":")
			yamlPosParent := getYamlPosition(lines, row, idxColon, false)
			yamlPosParentLastChild := yamlPosParent
			for yamlPosParentLastChild.child != nil {
				yamlPosParentLastChild = yamlPosParentLastChild.child
			}
			yamlPosParentLastChild.child = yamlPos
			yamlPos = yamlPosParent
			break
		}
	}

	return yamlPos
}

func codeComplete(content string, row, col int, fileType string) []map[string]string {
	lines := strings.Split(content, "\n")
	for i, _ := range lines {
		// remove cr characters
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}
	compl := []map[string]string{}
	yamlPos := getYamlPosition(lines, row, col, true)
	logger.Info("yP:", yamlPos)
	logger.Info("yPk:", yamlPos.getKey())
	//compl = append(compl, newCompletion("---\n", "---", "Start yaml document"))
	complDescr := loader.GetTemplatedYaml("codeComplete").Def
	compDescrFt := complDescr[fileType]
	if compDescrFt != nil {
		for {
			levelMap := compDescrFt.(map[string]interface{})
			matchingKeys, perfectMatch := getMatchingKeys(levelMap, yamlPos.keyPrefix)
			if perfectMatch {
				keyData := dataxform.IAsSIMap(levelMap[yamlPos.keyPrefix])
				if yamlPos.child != nil {
					compDescrFt = levelMap[yamlPos.keyPrefix].(map[string]interface{})["children"]
					yamlPos = yamlPos.child
					continue
				}
				valueCompleterName := dataxform.SIMapGetByKeyAsString(keyData, "valueCompleter")
				if valueCompleterName != "" {
					logger.Info("valCompleter:", valueCompleterName)
					valueCompleter := valueCompleters[valueCompleterName]
					if valueCompleter != nil {
						compl = valueCompleter(yamlPos, compl)
						break
					}
				}
			}
			for _, matchingKey := range matchingKeys {
				keyData := levelMap[matchingKey].(map[string]interface{})
				compl = append(compl, newCompletion(matchingKey+": ", matchingKey, keyData["descr"].(string)))
			}
			break
		}
	}
	return compl
}

func jsCompleter(yamlPos *YamlPosition, compl []map[string]string) []map[string]string {
	compl = append(compl, newCompletion("js", "js", "js completer"))
	return compl
}

func getMatchingKeys(level map[string]interface{}, keyPrefix string) ([]string, bool) {
	matching := []string{}
	perfectMatch := false
	for k, _ := range level {
		if strings.HasPrefix(k, keyPrefix) {
			matching = append(matching, k)
			if k == keyPrefix {
				perfectMatch = true
			}
		}
	}
	return matching, perfectMatch
}

func newCompletion(value, caption, meta string) map[string]string {
	completion := map[string]string{}
	completion["value"] = value
	completion["caption"] = caption
	completion["meta"] = meta
	return completion
}
