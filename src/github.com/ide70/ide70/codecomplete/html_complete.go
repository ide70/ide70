package codecomplete

import (
	//"fmt"
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
	"regexp"
	"strings"
)

var reWord = regexp.MustCompile(`\w+`)

func htmlCompleter(yamlPos *YamlPosition, edContext *EditorContext, configData map[string]interface{}, compl []map[string]string) []map[string]string {
	code := yamlPos.valuePrefx
	logger.Info("code:", code+"|")
	
	fileAsTemplatedYaml := loader.GetTemplatedYaml("htmlCompleterConfig", "ide70/dcfg/")
	if fileAsTemplatedYaml == nil {
		return compl		
	}
	htmlConfig := fileAsTemplatedYaml.Def
	
	idxTagStart := strings.LastIndex(code, "<")
	idxTagEnd := strings.LastIndex(code, ">")
	if strings.HasSuffix(code, " ") && idxTagStart > idxTagEnd {
		tagName := reWord.FindString(code[idxTagStart+1:])
		compl = append(compl, completeAttr(tagName, htmlConfig)...)
		return compl
	}
	
	tags:= dataxform.SIMapGetByKeyAsList(htmlConfig, "tags")
	for _,tagIf := range tags {
		tagData := dataxform.IAsSIMap(tagIf)
		tagName := dataxform.SIMapGetByKeyAsString(tagData, "name")
		tagDescr := dataxform.SIMapGetByKeyAsString(tagData, "descr")
		tagCaption := "<" + tagName + ">";
		tagValue := tagCaption + "</" + tagName + ">";
		compl = append(compl, newCompletion(tagValue, tagCaption, tagDescr))
	}
	return compl
}

func completeAttr(tagName string, htmlConfig map[string]interface{}) []map[string]string {
	compl := []map[string]string{}
	attrs:= dataxform.SIMapGetByKeyAsList(htmlConfig, "attrs")
	for _,attrIf := range attrs {
		attrData := dataxform.IAsSIMap(attrIf)
		if dataxform.SIMapGetByKeyIsString(attrData, "scope") {
			if dataxform.SIMapGetByKeyAsString(attrData, "scope") != "all" {
				continue;
			}
		} else {
			match := false
			attrScope := dataxform.SIMapGetByKeyAsList(attrData, "scope")
			for _,scopeIf := range attrScope {
				scopeTag := dataxform.IAsString(scopeIf)
				if tagName == scopeTag {
					match = true
				}
			}
			if !match {
				continue;
			}
		}
		attrName := dataxform.SIMapGetByKeyAsString(attrData, "name")
		attrDescr := dataxform.SIMapGetByKeyAsString(attrData, "descr")
		attrType := dataxform.SIMapGetByKeyAsString(attrData, "type")
		attrCaption := attrName;
		attrValue := attrName
		if attrType != "boolean" {
		 attrValue += `=""`;
		}
		compl = append(compl, newCompletion(attrValue, attrCaption, attrDescr))
	}
	return compl
}
