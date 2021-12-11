package codecomplete

import (
	"fmt"
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
	"regexp"
	"strings"
)

type NavigationResult struct {
	FileName string `json:"fileName"`
	Col      int    `json:"col"`
	Row      int    `json:"row"`
	Success  bool   `json:"success"`
}

func CodeNavigate(content string, row, col int, fileType string) *NavigationResult {
	lines := preProcessLines(content)

	yamlPos := getYamlPosition(lines, row, col, true)
	logger.Info("n.pos:", yamlPos.getKey())

	complDescr := loader.GetTemplatedYaml("codeNavigate", "").Def
	patternList := dataxform.SIMapGetByKeyAsList(complDescr, fileType)
	for _, patternIf := range patternList {
		pattern := dataxform.AsSIMap(patternIf)
		pathExpr := dataxform.SIMapGetByKeyAsString(pattern, "pathExpr")
		rePathExpr, _ := convertYamlpathToRegex(pathExpr, yamlPos)
		if !rePathExpr.MatchString(yamlPos.getKey()) {
			continue
		}
		logger.Info("navigation pathExpr metches:", pathExpr)
		fileNameSrc := dataxform.SIMapGetByKeyAsString(pattern, "fileName")
		fileName := ""
		targetValue := findTargetValue(lines[row], col)
		if fileNameSrc == "value" {
			fileName = targetValue
			logger.Info("navigation target value:", fileName)
		}
		addPrefix := dataxform.SIMapGetByKeyAsString(pattern, "addPrefix")
		addSuffix := dataxform.SIMapGetByKeyAsString(pattern, "addSuffix")
		if fileName != "" {
			fileName = "ide70/" + addPrefix + fileName + addSuffix
		}
		logger.Info("result file name:", fileName)

		navigateExpr := dataxform.SIMapGetByKeyAsString(pattern, "navigateTo")
		if navigateExpr != "" {
			var fileAsTemplatedYaml *loader.TemplatedYaml
			if fileName == "" {
				fileAsTemplatedYaml = loader.ConvertTemplatedYaml([]byte(content), "self")
			} else {
				fileNameParts := strings.Split(fileName, "/")
				fnPre := strings.Join(fileNameParts[:2], "/")
				fnPost := strings.Join(fileNameParts[2:], "/")
				fileAsTemplatedYaml = loader.GetTemplatedYaml(fnPost, fnPre+"/")
			}

			if fileAsTemplatedYaml != nil {
				fileData := fileAsTemplatedYaml.IDef
				logger.Info("pathExpr:", pathExpr)
				rePath, isValue := convertYamlpathToRegex(navigateExpr, yamlPos)
				if rePath != nil {
					row := 0
					col := 0
					dataxform.IApplyFn(fileData, func(entry dataxform.CollectionEntry) {
						logger.Info("leaf:", entry.LinearKey())
						if rePath.MatchString(entry.LinearKey()) {
							logger.Info("match")
							targetMatch := isValue && dataxform.IAsString(entry.Value()) == targetValue || !isValue && entry.Key() == targetValue
							if targetMatch {
								logger.Info("targetMatch:", entry.LinearKey())
								row, col = leafPos(entry.LinearKey(), lines)
								logger.Info("row, col:", row, col)
							}
						}
					})
					if row > 0 {
						return &NavigationResult{FileName: fileName, Row: row, Col: col, Success: true}
					}
					logger.Info("fileAsTemplatedYaml finished")
				}

			}
		}
		return &NavigationResult{FileName: fileName, Success: true}
	}

	//return &NavigationResult{FileName: "ide70/comp/layer/layer.yaml", Col: 1, Row: 3, Success: true}
	logger.Info("no matching navigation found")
	return &NavigationResult{Success: false}
}

var rePath = regexp.MustCompile(`[/.\w]+`)

func findTargetValue(line string, col int) string {
	startIdx := strings.LastIndexAny(line[:col], " ")
	if startIdx == -1 {
		startIdx = 0
	}
	return rePath.FindString(line[startIdx:])
}

func leafPos(path string, lines []string) (row, col int) {
	pathTokens := strings.Split(path, ".")
	row = 0
	col = 0
	tokenNo := 0
	idxToSearch := -1
	for row < len(lines) {
		line := lines[row]
		if strings.HasSuffix(line, "---") {
			logger.Info("skipping --- at row", row)
			row++
			continue
		}
		lineFromCol := line[col:]
		if idxToSearch > -1 {
			if strings.HasPrefix(lineFromCol, "- ") {
				logger.Info("idx entry at line:", row)
				idxToSearch--
				if idxToSearch == -1 {
					logger.Info("idx found at line:", row)
					if tokenNo < len(pathTokens)-1 {
						tokenNo++
						col += 2
						continue
					} else {
						// index found
						return
					}
				}
			}
			row++
			continue
		}

		token := pathTokens[tokenNo]
		if strings.HasPrefix(token, "[") {
			idxStr := strings.TrimSuffix(strings.TrimPrefix(token, "["), "]")
			fmt.Sscanf(idxStr, "%d", &idxToSearch)
			logger.Info("idxToSearch:", idxToSearch)
			continue
		} else {
			if strings.HasPrefix(lineFromCol, token+":") {
				logger.Info("key", token, "found at line:", row)
				if tokenNo < len(pathTokens)-1 {
					tokenNo++
					col += 2
					row++
					continue
				} else {
					// key found
					return
				}
			}
		}
		row++
	}
	return 0, 0
}
