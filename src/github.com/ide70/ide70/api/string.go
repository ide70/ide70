package api

import (
	"strings"
)

type String struct {
	s string
}

func (sW String) EndsWith(suffixIf interface{}) bool {
	s := sW.s
	suffix := IAsString(suffixIf)
	return strings.HasSuffix(string(s), suffix)
}

func (sW String) startsWith(suffixIf interface{}) bool {
	s := sW.s
	suffix := IAsString(suffixIf)
	return strings.HasPrefix(string(s), suffix)
}

func (sW String) S() string {
	return sW.s
}
