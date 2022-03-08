package api

import ()

type Arrays struct {
}

type ITableW struct {
	t ITable
}

func (a *API) Arrays() *Arrays {
	return &Arrays{}
}

func (as *Arrays) NewITableW() *ITableW {
	return &ITableW{t: ITable{}}
}

func (tw *ITableW) AddCol(col string, v interface{}) *ITableW {
	row := ensureRow(tw)
	tw.t[row][col] = v
	return tw
}

func (tw *ITableW) AddEmptyRow() *ITableW {
	tw.t = append(tw.t, SIMap{})
	return tw
}

func ensureRow(tw *ITableW) int {
	if len(tw.t) == 0 {
		tw.t = append(tw.t, SIMap{})
	}
	return len(tw.t) - 1
}

func (tw *ITableW) Finalize() ITable {
	logger.Info("finalize, length:", len(tw.t))
	return tw.t
}
