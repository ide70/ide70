package file

import (
	"github.com/ide70/ide70/util/log"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

var logger = log.Logger{"file"}

type FileContext struct {
}

func (fc *FileContext) GetLastPathTag(path string) string {
	tokens := strings.Split(path, "/")
	return tokens[len(tokens)-1]
}

func (fc *FileContext) TrimLastPathTag(path string) string {
	tokens := strings.Split(path, "/")
	return strings.Join(tokens[0:len(tokens)-1], "/")
}

func (fc *FileContext) AppendPath(path, tag string) string {
	tokens := strings.Split(path, "/")
	tokens = append(tokens, tag)
	return strings.Join(tokens, "/")
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

func (fc *FileContext) CreateFolderAll(path string) {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		logger.Error(err.Error())
	}
}

func (fc *FileContext) CreateFileWithPath(path string) {
	dirPath := fc.TrimLastPathTag(path)
	fc.CreateFolderAll(dirPath)
	fc.CreateFile(path)
}

func (fc *FileContext) IsRegularFile(path string) bool {
	sourceFileStat, err := os.Stat(path)
	if err != nil {
		return false
	}

	return sourceFileStat.Mode().IsRegular()
}

func (fc *FileContext) RemoveAll(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		logger.Error(err.Error())
	}
}

func (fc *FileContext) Remove(path string) {
	err := os.Remove(path)
	if err != nil {
		logger.Error(err.Error())
	}
}

func (fc *FileContext) Move(oldpath, newpath string) {
	err := os.Rename(oldpath, newpath)
	if err != nil {
		logger.Error(err.Error())
	}
}

func (fc *FileContext) Copy(src, dst string) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	if !sourceFileStat.Mode().IsRegular() {
		logger.Error("%s is not a regular file", src)
		return
	}

	source, err := os.Open(src)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	if err != nil {
		logger.Error(err.Error())
	}
}

func FileList(basePath string, trimPrefix string, trimSuffix string) []string {
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
			list = append(list, FileList(childPath, trimPrefix, trimSuffix)...)
			continue
		}

		list = append(list, strings.TrimSuffix(strings.TrimPrefix(childPath, trimPrefix), trimSuffix))
	}
	return list
}

func DirList(basePath string, trimPrefix string) []string {
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
			list = append(list, strings.TrimPrefix(childPath, trimPrefix))
			list = append(list, DirList(childPath, trimPrefix)...)
			continue
		}

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
