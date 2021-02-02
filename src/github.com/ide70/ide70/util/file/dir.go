package file

import (
	"github.com/ide70/ide70/util/log"
	"io/ioutil"
	"os"
	"strings"
)

var logger = log.Logger{"file"}

func FileList(basePath string, trimPrefix string) []string {
	list := []string{}
	files, _ := ioutil.ReadDir(basePath)
	for _, afile := range files {
		childPath := basePath + "/" + afile.Name()

		stat, err := os.Stat(childPath)

		if err != nil {
			logger.Error("stat failed:", childPath)
			continue
		}
		isFolder := stat.IsDir()
		if isFolder {
			list = append(list, FileList(childPath, trimPrefix)...)
			continue
		}

		list = append(list, strings.TrimPrefix(childPath, trimPrefix))
	}
	return list
}

func CompactFileList(basePath string) string {
	builder := strings.Builder{}
	compactFileList(basePath, "", &builder)
	return builder.String()
}

func compactFileList(basePath string, name string, list *strings.Builder) {
	list.WriteString("[")
	first := true
	if name != "" {
		addFileName(name, list)
		first = false
	}
	files, _ := ioutil.ReadDir(basePath)
	for _, afile := range files {
		if !first {
			list.WriteString(",")
		} else {
			first = false
		}
		childPath := basePath + "/" + afile.Name()

		stat, err := os.Stat(childPath)
		if err != nil {
			logger.Error("stat failed:", childPath)
			continue
		}

		if stat.IsDir() {
			compactFileList(childPath, afile.Name(), list)
			continue
		}
		addFileName(afile.Name(), list)
	}
	list.WriteString("]")
}
//strings.TrimPrefix(basePath, trimPrefix)

func addFileName(name string, list *strings.Builder) {
	list.WriteString("\"")
	list.WriteString(name)
	list.WriteString("\"")
}
