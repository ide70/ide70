package server

import (
	"sync/atomic"
	"github.com/ide70/ide70/comp"
	"fmt"
)

type UnitCache struct {
	Seq int64
	ActiveUnits map[string]*comp.UnitRuntime
}

func newUnitCache() *UnitCache {
	return &UnitCache{ActiveUnits: map[string]*comp.UnitRuntime{}}
}

func (unitCache *UnitCache) addUnit(unit *comp.UnitRuntime) {
	newSeq := atomic.AddInt64(&unitCache.Seq, 1)
	newSeqStr := fmt.Sprintf("u%d",newSeq)
	unit.AssignID(newSeqStr)
	unitCache.ActiveUnits[newSeqStr] = unit
}