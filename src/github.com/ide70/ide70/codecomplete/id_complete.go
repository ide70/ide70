package codecomplete

import (
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/loader"
	"strings"
)

// yamlPath expression
// key1.*

func idCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	srcExpr1 := api.SIMapGetByKeyAsString(configData, "srcExpr1")
	srcExpr2 := api.SIMapGetByKeyAsString(configData, "srcExpr2")
	fileAsTemplatedYaml := loader.ConvertTemplatedYaml([]byte(edContext.content), "self")

	if fileAsTemplatedYaml != nil {
		fileData := fileAsTemplatedYaml.IDef
		logger.Debug("srcExpr1:", srcExpr1)
		srcVal1 := firstMatchValue(srcExpr1, yamlPos, fileData)
		if srcExpr2 != "" {
			srcVal1 += " " + firstMatchValue(srcExpr2, yamlPos, fileData)
		}
		srcVal1 = idCreate(srcVal1)

		compl = addCompletion(srcVal1, edContext, configData, compl)

	}

	return compl
}

var autoRelpace = map[string]string{
	"á": "a",
	"é": "e",
	"í": "i",
	"ó": "o",
	"ö": "o",
	"ő": "o",
	"ú": "u",
	"ü": "u",
	"ű": "u",
	"Á": "A",
	"É": "E",
	"Í": "I",
	"Ó": "O",
	"Ö": "O",
	"Ő": "O",
	"Ú": "U",
	"Ü": "U",
	"Ű": "U",
	"!": "",
}

func idCreate(s string) string {
	s = splitCapitalize(s, " ")
	s = splitCapitalize(s, "/")
	for f, t := range autoRelpace {
		s = strings.ReplaceAll(s, f, t)
	}

	return s
}

func splitCapitalize(s, by string) string {
	tokens := strings.Split(s, by)
	for idx := range tokens {
		if idx > 0 {
			tokens[idx] = strings.Title(tokens[idx])
		}
	}
	return strings.Join(tokens, "")
}

func firstMatchValue(expr string, yamlPos *YamlPosition, data interface{}) string {
	value := ""
	re, isValue := convertYamlpathToRegex(expr, yamlPos)
	api.IApplyFn(data, func(entry api.CollectionEntry) {
		logger.Debug("leaf:", entry.LinearKey())
		if re.MatchString(entry.LinearKey()) {
			logger.Debug("match")
			value, _ = leafValDescr(entry, isValue)
			entry.Stop()
		}
	})
	return value
}
