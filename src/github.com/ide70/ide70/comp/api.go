package comp

import (
	"github.com/ide70/ide70/dataxform"
)

type SIMap map[string]interface{}

type Interface struct {
	I interface{}
}

func (i Interface) AsMap() SIMap {
	return dataxform.IAsSIMap(i.I)
}

func (i Interface) AsString() string {
	return dataxform.IAsString(i.I)
}

func (m SIMap) Delete(key string) {
	if m == nil {
		return
	}
	delete(m, key)
}

func (m SIMap) Put(key string, value interface{}) {
	if m == nil {
		return
	}
	m[key] = value
}

func (m SIMap) Get(key string) Interface {
	if m == nil {
		return Interface{nil}
	}
	return Interface{m[key]}
}
