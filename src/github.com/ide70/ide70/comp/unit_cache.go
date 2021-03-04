package comp

import (
	"fmt"
	"sync/atomic"
)

type UnitCache struct {
	Seq         int64
	ActiveUnits map[string]*UnitRuntime
}

func NewUnitCache() *UnitCache {
	return &UnitCache{ActiveUnits: map[string]*UnitRuntime{}}
}

func (unitCache *UnitCache) AddUnit(unit *UnitRuntime) {
	newSeq := atomic.AddInt64(&unitCache.Seq, 1)
	newSeqStr := fmt.Sprintf("u%d", newSeq)
	unit.AssignID(newSeqStr)
	unitCache.ActiveUnits[newSeqStr] = unit
}
