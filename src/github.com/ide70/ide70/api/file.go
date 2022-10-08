package api

import (
	"os"
	"io/ioutil"
)

type File struct {
	fullPath string
}

func NewFile(fullPath string) *File {
	return &File{fullPath: fullPath}
}

func (f *File) AppendText(text string) *File {
	osf, err := os.OpenFile(f.fullPath,
		os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return f
	}
	defer osf.Close()
	if _, err = osf.WriteString(text); err != nil {
		logger.Error(err.Error())
	}
	return f
}

func (f *File) WriteBinaryData(data *BinaryData) *File {
	ioutil.WriteFile(f.fullPath, *data.GetData(), 0644)
	return f
}