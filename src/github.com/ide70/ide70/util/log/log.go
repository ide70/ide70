package log

import (
	"fmt"
	"strings"
	"time"
)

const TRACE int = 0
const DEBUG int = 1
const INFO int = 2
const WARNING int = 3
const ERROR int = 4
const OFF int = 5
const FORCE int = 6

var levelNames = []string{"TRACE", "DEBUG", "INFO", "WARNING", "ERROR", "OFF"}
var levelNamesInLog = []string{"", "", "", "WARNING: ", "ERROR: ", "", ""}
var Prefix string
var TTY bool = false

var keyMap map[string]int = map[string]int{"": DEBUG}

type Logger struct {
	Key string
}

func (logger *Logger) Trace(value ...interface{}) {
	TraceKey(logger.Key, value...)
}

func (logger *Logger) Debug(value ...interface{}) {
	DebugKey(logger.Key, value...)
}

func (logger *Logger) Info(value ...interface{}) {
	InfoKey(logger.Key, value...)
}

func (logger *Logger) Warning(value ...interface{}) {
	WarningKey(logger.Key, value...)
}

func (logger *Logger) Error(value ...interface{}) {
	ErrorKey(logger.Key, value...)
}

func (logger *Logger) Force(value ...interface{}) {
	ForceKey(logger.Key, value...)
}

func (logger *Logger) GetLevel() int {
	keyLevel, ok := keyMap[logger.Key]
	if ok {
		return keyLevel
	}
	keyLevel, _ = keyMap[""]
	return keyLevel
}

func Trace(value ...interface{}) {
	log(TRACE, "", value...)
}

func Debug(value ...interface{}) {
	log(DEBUG, "", value...)
}

func Info(value ...interface{}) {
	log(INFO, "", value...)
}

func Warning(value ...interface{}) {
	log(WARNING, "", value...)
}

func Error(value ...interface{}) {
	log(ERROR, "", value...)
}

func TraceKey(key string, value ...interface{}) {
	log(TRACE, key, value...)
}

func DebugKey(key string, value ...interface{}) {
	log(DEBUG, key, value...)
}

func InfoKey(key string, value ...interface{}) {
	log(INFO, key, value...)
}

func WarningKey(key string, value ...interface{}) {
	log(WARNING, key, value...)
}

func ErrorKey(key string, value ...interface{}) {
	log(ERROR, key, value...)
}

func ForceKey(key string, value ...interface{}) {
	log(FORCE, key, value...)
}

func visible(level int, key string) bool {
	if keyLevel, ok := keyMap[key]; ok {
		return level >= keyLevel
	}
	if key == "" {
		return false
	}
	globLevel, ok := keyMap[""]
	return ok && level >= globLevel
}

func log(level int, key string, value ...interface{}) {
	if visible(level, key) {
		switch valueT := value[0].(type) {
		case string:
			if strings.Index(valueT, "\n") != -1 {
				logMultiline(level, valueT)
				return
			}
		}
		fmt.Print(time.Now().Format("[2006-01-02 15:04:05] "))
		fmt.Print(levelNamesInLog[level])
		fmt.Print(Prefix)
		fmt.Println(value...)
	}
}

func logMultiline(level int, s string) {
	lines := strings.Split(s, "\n")
	timeStr := time.Now().Format("[2006-01-02 15:04:05] ")
	for _, line := range lines {
		fmt.Print(timeStr)
		fmt.Print(levelNamesInLog[level])
		fmt.Print(Prefix)
		fmt.Println(line)
	}
}

func SetKeyLevel(key string, level int) {
	keyMap[key] = level
}

func GetKeyLevel(key string) int {
	keyLevel, ok := keyMap[key]
	if ok {
		return keyLevel
	}
	keyLevel, _ = keyMap[""]
	return keyLevel
}

func LevelToString(level int) string {
	return levelNames[level]
}
