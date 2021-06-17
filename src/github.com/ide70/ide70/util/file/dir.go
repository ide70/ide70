package file

import (
	"github.com/ide70/ide70/util/log"
	"io/ioutil"
	"os"
	"strings"
)

var logger = log.Logger{"file"}

type FileContext struct {
}

func (fc *FileContext) ReadDir(basePath string) []interface{} {
	list := []interface{}{}
	files, _ := ioutil.ReadDir(basePath)
	for _, afile := range files {
		childPath := basePath + "/" + afile.Name()

		stat, err := os.Stat(childPath)

		if err != nil {
			logger.Error("stat failed:", childPath)
			continue
		}
		entry := map[string]interface{}{}
		entry["name"] = afile.Name()
		entry["path"] = childPath
		entry["isDir"] = stat.IsDir()
		list = append(list, entry)
	}
	return list
}

func (fc *FileContext) CreateFile(path string) {
	emptyFile, err := os.Create(path)
	if err != nil {
		logger.Error(err.Error())
	}
	emptyFile.Close()
}

func (fc *FileContext) CreateFolder(path string) {
	err := os.Mkdir(path, 0755)
	if err != nil {
		logger.Error(err.Error())
	}
}

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
