package server

import (
	"encoding/json"
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
	parent      *YamlPosition
}

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

	completions := codeComplete(content, int(row), int(col))
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(completions)
}

func (yPos *YamlPosition) getKey() string {
	key := yPos.keyPrefix
	if yPos.parent != nil {
		key = yPos.parent.getKey() + "." + key
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

func getYamlPosition(lines []string, row, col int) *YamlPosition {
	yamlPos := &YamlPosition{}
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
			if len(tokens) > 1 {
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
			yamlPos.parent = getYamlPosition(lines, row, idxColon)
			break
		}
	}

	return yamlPos
}

func codeComplete(content string, row, col int) []map[string]string {
	lines := strings.Split(content, "\n")
	compl := []map[string]string{}
	yamlPos := getYamlPosition(lines, row, col)
	logger.Info("yP:", yamlPos)
	logger.Info("yPk:", yamlPos.getKey())
	compl = append(compl, newCompletion("---\n", "---", "Start yaml document"))
	return compl
}

func newCompletion(value, caption, meta string) map[string]string {
	completion := map[string]string{}
	completion["value"] = value
	completion["caption"] = caption
	completion["meta"] = meta
	return completion
}
