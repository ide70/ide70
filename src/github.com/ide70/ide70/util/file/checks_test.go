/*******************************************************************************
* checks_test.go
* 2016.03.17 - Base version                                                  AN
*******************************************************************************/
package file


import (
	"testing"
	"os"
	"io/ioutil"
	"path/filepath"
    "reflect"
    "runtime"
)


func GetFunctionName(i interface{}) string {
    return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

type Creator interface {
	Create(path string) string
}

type File struct {
	name string
	perm os.FileMode
}
func (file File) Create(path string) string {
	name := path + "/" + file.name
	ioutil.WriteFile(name, []byte("test"), file.perm)
	os.Chmod(name, file.perm) // hack
	return name
}

type Dir struct {
	name string
	perm os.FileMode
}
func (dir Dir) Create(path string) string {
	name := path + "/" + dir.name
	os.MkdirAll(name, dir.perm)
	os.Chmod(name, dir.perm) // hack
	return name
}

type Link struct {
	create	interface{}
	name 	string
}
func (link Link) Create(path string) string {
	oldname := "/invalid_source"
	if link.create != nil {
		oldname = link.create.(Creator).Create(path)
	}
	os.Symlink(oldname, path + "/" + link.name)
	return path + "/" + link.name
}


func TestChecksBase(t *testing.T) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		t.Error(err)
		return
    }
	// dir = "."

	var tests = []struct {
		testFunc	func(string)(bool)
		fileName	string
		result		bool
		create		interface{}
	} {
		{CanRead, "",             false, nil},
		{CanRead, ".",            true , nil},
		{CanRead, "*",            false, nil},
		{CanRead, "/",            true , nil},
		{CanRead, "CanReadFile1", false, File{"CanReadFile1", 0111} },
		{CanRead, "CanReadFile2", true , File{"CanReadFile2", 0711} },
		{CanRead, "CanReadFile3", true , File{"CanReadFile3", 0171} },
		{CanRead, "CanReadDir1",  false, Dir{"CanReadDir1", 0111} },
		{CanRead, "CanReadDir2",  true , Dir{"CanReadDir2", 0711} },
		{CanRead, "CanReadDir3",  true , Dir{"CanReadDir3", 0171} },

		{CanWrite, "",  false, nil},
		{CanWrite, ".", true , nil},
		{CanWrite, "*", false, nil},
		{CanWrite, "/", false, nil},
		{CanWrite, "CanWriteFile1", false, File{"CanWriteFile1", 0111} },
		{CanWrite, "CanWriteFile2", true , File{"CanWriteFile2", 0711} },
		{CanWrite, "CanWriteFile3", true , File{"CanWriteFile3", 0171} },
		{CanWrite, "CanWriteFile4", true , File{"CanWriteFile4", 0060} },
		{CanWrite, "CanWriteFile5", true , File{"CanWriteFile5", 0222} },

		{IsNormalFile, "",  false, nil},
		{IsNormalFile, ".", false, nil},
		{IsNormalFile, "*", false, nil},
		{IsNormalFile, "/", false, nil},
		{IsNormalFile, "IsNormalFileFile1", true,  File{"IsNormalFileFile1", 0111} },
		{IsNormalFile, "IsNormalFileFile2", true , File{"IsNormalFileFile2", 0711} },
		{IsNormalFile, "IsNormalFileFile3", true , File{"IsNormalFileFile3", 0171} },
		{IsNormalFile, "IsNormalFileFile4", true , File{"IsNormalFileFile4", 0060} },
		{IsNormalFile, "IsNormalFileFile5", true , File{"IsNormalFileFile5", 0222} },
		{IsNormalFile, "IsNormalFileLink1", false, Link{File{"IsNormalFileFile6", 0222}, "IsNormalFileLink1" } },
		{IsNormalFile, "IsNormalFileLink2", false, Link{Dir{"IsNormalFileDir1", 0222},   "IsNormalFileLink2" } },
		{IsNormalFile, "IsNormalFileLink3", false, Link{nil,                             "IsNormalFileLink3" } },

		{IsDir, "",           false, nil},
		{IsDir, ".",          true , nil},
		{IsDir, "*",          false, nil},
		{IsDir, "/",          true , nil},
		{IsDir, "IsDirFile1", false, File{"IsDirFile1", 0222} },
		{IsDir, "IsDirDir1",  true,  Dir{"IsDirDir1", 0171} },
		{IsDir, "IsDirLink1", false, Link{File{"IsDirFile2", 0222}, "IsDirLink1" } },
		{IsDir, "IsDirLink2", false, Link{Dir{"IsDirDir2", 0222},   "IsDirLink2" } },
		{IsDir, "IsDirLink3", false, Link{nil,                      "IsDirLink3" } },

		{IsExist, "",             false, nil},
		{IsExist, ".",            true , nil},
		{IsExist, "*",            false, nil},
		{IsExist, "/",            true , nil},
		{IsExist, "IsExistFile1", true , File{"IsExistFile1", 0222} },
		{IsExist, "IsExistDir1",  true , Dir{"IsExistDir1", 0171} },
		{IsExist, "IsExistLink1", true, Link{File{"IsExistFile2", 0222}, "IsExistLink1" } },
		{IsExist, "IsExistLink2", true, Link{Dir{"IsExistDir2", 0222},   "IsExistLink2" } },
		{IsExist, "IsExistLink3", true, Link{nil,                        "IsExistLink3" } },

		{IsSymlink, "",  false, nil},
		{IsSymlink, ".", false, nil},
		{IsSymlink, "*", false, nil},
		{IsSymlink, "/", false, nil},
		{IsSymlink, "IsSymlinkFile1", false , File{"IsSymlinkFile1", 0222} },
		{IsSymlink, "IsSymlinkDir1",  false , Dir{"IsSymlinkDir1", 0171} },
		{IsSymlink, "IsSymlinkLink1", true, Link{File{"IsSymlinkFile2", 0222}, "IsSymlinkLink1" } },
		{IsSymlink, "IsSymlinkLink2", true, Link{Dir{"IsSymlinkDir2", 0222},   "IsSymlinkLink2" } },
		{IsSymlink, "IsSymlinkLink3", true, Link{nil,                          "IsSymlinkLink3" } },

		{IsNormalFileAbs, "",  false, nil},
		{IsNormalFileAbs, ".", false, nil},
		{IsNormalFileAbs, "*", false, nil},
		{IsNormalFileAbs, "/", false, nil},

		{IsNormalFileAbs, "IsNormalFileAbsFile1", true , File{"IsNormalFileAbsFile1", 0222} },
		{IsNormalFileAbs, "IsNormalFileAbsDir1",  false , Dir{"IsNormalFileAbsDir1", 0171} },
		{IsNormalFileAbs, "IsNormalFileAbsLink1", true, Link{File{"IsNormalFileAbsFile2", 0222}, "IsNormalFileAbsLink1" } },
		{IsNormalFileAbs, "IsNormalFileAbsLink2", false, Link{Dir{"IsNormalFileAbsDir2", 0222},   "IsNormalFileAbsLink2" } },
		{IsNormalFileAbs, "IsNormalFileAbsLink3", false, Link{nil,                                "IsNormalFileAbsLink3" } },
	}
	for _, test := range tests {
		testFileName := test.fileName

		if test.create != nil {
			testFileName = dir + "/" + test.fileName
			test.create.(Creator).Create(dir)
		}

		result := test.testFunc(testFileName)
		if result != test.result {
			t.Error(GetFunctionName(test.testFunc) + "(\"" + testFileName + "\") =", result)
		}

	}
}
