package codecomplete

import (
	"fmt"
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
	"github.com/ide70/ide70/util/file"
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

	complDescrTY := loader.GetTemplatedYaml("codeNavigate", "")
	if complDescrTY == nil {
		return &NavigationResult{Success: false}
	}
	complDescr := complDescrTY.Def
	patternList := dataxform.SIMapGetByKeyAsList(complDescr, fileType)
	for _, patternIf := range patternList {
		pattern := dataxform.AsSIMap(patternIf)
		pathExpr := dataxform.SIMapGetByKeyAsString(pattern, "pathExpr")
		rePathExpr, _ := convertYamlpathToRegex(pathExpr, yamlPos)
		if !rePathExpr.MatchString(yamlPos.getKey()) {
			continue
		}
		logger.Info("navigation pathExpr matches:", pathExpr)
		fileNameSrc := dataxform.SIMapGetByKeyAsString(pattern, "fileName")
		fileName := ""
		createFile := false
		targetValue, preceeding := findTargetValue(lines[row], col)
		preceedingRE := dataxform.SIMapGetByKeyAsString(pattern, "preceedingRE")
		if preceedingRE != "" {
			rePreceeding ,err := regexp.Compile(preceedingRE)
			if err != nil {
				logger.Warning("skipping invalid RE:", preceedingRE)
				continue
			}
			if !rePreceeding.MatchString(preceeding) {
				logger.Warning("preceeding not match:", preceeding)
				continue
			}
		}
		logger.Info("targetValue:", targetValue)
		if fileNameSrc == "value" {
			fileName = targetValue
			if strings.HasSuffix(fileName, "+") {
				createFile = true
				fileName = strings.TrimSuffix(fileName, "+")
			}
			logger.Info("navigation target value:", fileName)
		}
		addPrefix := dataxform.SIMapGetByKeyAsString(pattern, "addPrefix")
		addSuffix := dataxform.SIMapGetByKeyAsString(pattern, "addSuffix")
		if fileName != "" {
			fileName = "ide70/" + addPrefix + fileName + addSuffix
		}
		logger.Info("result file name:", fileName)
		
		if fileName != "" && !createFile {
			logger.Info("checking target file: ", fileName)
			fc := &file.FileContext{}
			if !fc.IsRegularFile(fileName) {
				logger.Info("missing target file: ", fileName)
				return &NavigationResult{Success: false}
			}
		}

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
		if fileName != "" && createFile {
			logger.Info("checking create new file: ", fileName)
			fc := &file.FileContext{}
			if !fc.IsRegularFile(fileName) {
				fc.CreateFileWithPath(fileName)
				logger.Info("create new file: ", fileName)
			}
		}
		return &NavigationResult{FileName: fileName, Success: true}
	}

	//return &NavigationResult{FileName: "ide70/comp/layer/layer.yaml", Col: 1, Row: 3, Success: true}
	logger.Info("no matching navigation found")
	return &NavigationResult{Success: false}
}

var rePath = regexp.MustCompile(`[/.\w+]+`)

func findTargetValue(line string, col int) (value, preceeding string) {
	startIdx := strings.LastIndexAny(line[:col], " (\"")
	if startIdx == -1 {
		startIdx = 0
	}
	value = rePath.FindString(line[startIdx:])
	preceeding = line[:startIdx]
	if strings.HasSuffix(preceeding, "\"") {
		preceeding = strings.TrimSuffix(preceeding, "\"")
	}
	return
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
