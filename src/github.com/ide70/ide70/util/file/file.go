package file

import (
	"errors"
	"os"
	"strings"
	"path/filepath"
)

const maxDepth = 40

func LastPathComponent(path string) string {
	tokens := strings.Split(path, "/")
	return tokens[len(tokens)-1]
}

func TrimLastPathComponent(path string) string {
	tokens := strings.Split(path, "/")
	return strings.Join(tokens[0:len(tokens)-1], "/")
}

func FirstPathComponent(path string) string {
	tokens := strings.Split(path, "/")
	return tokens[0]
}

func PrefixFilename(path string, prefix string) string {
	tokens := strings.Split(path, "/")
	return strings.Join(tokens[0:len(tokens)-1], "/") + "/" + prefix + tokens[len(tokens)-1]
}

// resolve nested links deeper than system limit
func ResolveLink(path string) (string, error) {
	return filepath.EvalSymlinks(path)
	//pathLast := LastPathComponent(path)
	//pathPrefix := strings.TrimSuffix(path, "/"+pathLast)
	//return resolveLink(pathPrefix, pathLast, 0)
}

func resolveLink(abspath string, linkpath string, depth int) (string, error) {
	if depth > maxDepth {
		return "", errors.New("max link depth exceeded")}
	linkpathFirst := FirstPathComponent(linkpath)
	linkpathRest := strings.TrimPrefix(linkpath, linkpathFirst)
	path := abspath + "/" + linkpathFirst
	fi, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return path, nil
	}
	dest, err := os.Readlink(path)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(dest, "/") {
		return dest + linkpathRest, nil
	} else {
		return resolveLink(abspath, dest+linkpathRest, depth+1)
	}
}
