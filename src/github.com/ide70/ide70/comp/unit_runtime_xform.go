package comp

import ()

type UnitXform struct {
	unit       *UnitRuntime
	unitImport *UnitRuntime
	mountComp  *CompRuntime
}

func (uCtx *UnitCtx) CreateXForm(importUnitName string) *UnitXform {
	unit := uCtx.unit
	unitImport := InstantiateUnit(importUnitName, unit.Application, unit.appParams, unit.PassContext)
	if unitImport == nil {
		return nil
	}
	return &UnitXform{unit: unit, unitImport: unitImport, mountComp: unit.RootComp}
}

func (uXf *UnitXform) SetMountComp(mountComp *CompRuntime) {
	uXf.mountComp = mountComp
}

func (uXf *UnitXform) Import() {
	unitImport := uXf.unitImport
	unit := uXf.unit
	for _, comp := range unitImport.CompByChildRefId {
		if comp == unitImport.RootComp {
			continue
		}
		comp.Unit = unit
		unit.registerComp(comp)
	}
	for _, comp := range unitImport.RootComp.Children {
		uXf.mountComp.Children = append(uXf.mountComp.Children, comp)
	}
}
