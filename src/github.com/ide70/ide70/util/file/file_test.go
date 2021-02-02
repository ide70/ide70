/*******************************************************************************
* file_test.go
* 2016.03.17 - Base version                                                  AN
*******************************************************************************/
package file


import (
	"testing"
	"path/filepath"
	"os"
	"fmt"
	"os/exec"
)


func TestLastPathComponent(t *testing.T) {
	var tests = []struct {
		path	string
		result	string
	} {
		{"", ""},
		{"aha", "aha"},
		{"aha/baha", "baha"},
		{"aha//", ""},
		{"/", ""},
		{"//", ""},
	}
	for _, test := range tests {
		result := LastPathComponent(test.path)
		if  result != test.result {
			t.Error("LastPathComponent(\"" + test.path + "\") = " + result)
		}
	}
}


func TestTrimLastPathComponent(t *testing.T) {
	var tests = []struct {
		path	string
		result	string
	} {
		{"", ""},
		{"aha", ""},
		{"aha/baha", "aha"},
		{"aha//", "aha/"},
		{"/", ""},
		{"//", "/"},
	}
	for _, test := range tests {
		result := TrimLastPathComponent(test.path)
		if  result != test.result {
			t.Error("TrimLastPathComponent(\"" + test.path + "\") = " + result)
		}
	}
}

func TestResolveLink(t *testing.T) {
	var tests = []struct {
		path	string
		result	string
		error	error
	}{
		{"", ".", nil},
	}
	for _, test := range tests {
		result, error := ResolveLink(test.path)
		if  (result != test.result) || (error != test.error) {
			t.Error("ResolveLink(\"" + test.path + "\") = " + result)
		}
	}
}

func OsCommand(cmd string) {
	fmt.Println("OsCommand(" + cmd + ")")
	out, _ := exec.Command("bash", "-c", cmd).Output()
	outstr := string(out)
	if outstr != "" {
		fmt.Println(outstr)
	}
}

func TestResolveLinkPrivate(t *testing.T) {

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))

	os.MkdirAll(dir + "/testfolder", 0755)
	os.Symlink(dir + "/testfolder", dir + "/testlink")
	os.Symlink("testlink", dir + "/testlink2")
	os.Symlink("testlink3", dir + "/testlink3")

	var tests = []struct {
		abspath		string
		linkpath	string
		depth		int
		result		string
		error		string
	}{
		{"", "", 0, "/", ""},
		{"/dmsservicetest", "", 0, "", "lstat /dmsservicetest/: no such file or directory"},
		{dir, "", 0, dir + "/", ""},
		{dir, "testlink", 0, dir + "/testfolder", ""},
		{dir, "testlink2", 0, dir + "/testfolder", ""},
		{dir, "testlink3", 0, "", "max link depth exceeded"},

	}
	for _, test := range tests {
		result, error := resolveLink(test.abspath, test.linkpath, test.depth)
		if  (result != test.result) || (error != nil && error.Error() != test.error) {
			errorStr := ""
			if error != nil {
				errorStr = error.Error()
			}
			t.Error("resolveLink(\"" + test.abspath + ", " + test.linkpath + "," + string(test.depth) + "\") = " + result + "," + errorStr)
		}
	}
}
