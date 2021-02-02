package file

import (
	"os"
	//"syscall"
)

const other_rd os.FileMode = 1 << 2
const group_rd os.FileMode = 1 << 5
const user_rd os.FileMode = 1 << 8

const other_wr os.FileMode = 1 << 1
const group_wr os.FileMode = 1 << 4
const user_wr os.FileMode = 1 << 7

/*func CanRead(fileName string) bool {
	stat, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}
	fm := stat.Mode().Perm()
	if fm&other_rd != 0 {
		return true
	} else if (fm&group_rd) != 0 && (os.Getegid() == int(stat.Sys().(*syscall.Stat_t).Gid)) {
		return true
	} else if (fm&user_rd) != 0 && (os.Geteuid() == int(stat.Sys().(*syscall.Stat_t).Uid)) {
		return true
	}

	return false
}

func CanWrite(fileName string) bool {
	stat, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}
	if os.Geteuid() == 0 {
		return true
	}
	fm := stat.Mode().Perm()
	if fm&other_wr != 0 {
		return true
	} else if (fm&group_wr) != 0 && (os.Getegid() == int(stat.Sys().(*syscall.Stat_t).Gid)) {
		return true
	} else if (fm&user_wr) != 0 && (os.Geteuid() == int(stat.Sys().(*syscall.Stat_t).Uid)) {
		return true
	}

	return false
}*/

func IsSymlink(fileName string) bool {
	fi, err := os.Lstat(fileName)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSymlink != 0
}

func IsNormalFileAbs(fileName string) bool {
	fileName, _ = ResolveLink(fileName)
	return IsNormalFile(fileName)
}

func IsNormalFile(fileName string) bool {
	dest, err := os.Readlink(fileName)
	if dest != "" {
		return false
	}
	stat, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}
	return stat.Mode().IsRegular()
}

func IsDir(fileName string) bool {
	dest, err := os.Readlink(fileName)
	if dest != "" {
		return false
	}
	stat, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}
	return stat.Mode().IsDir()
}

func IsExist(fileName string) bool {
	dest, err := os.Readlink(fileName)
	if dest != "" {
		return true
	}
	_, err = os.Stat(fileName)
	return err == nil
}
