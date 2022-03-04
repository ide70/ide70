package api

import ()

type Arrays struct {
}

func (a *API) Arrays() *Arrays {
	return &Arrays{}
}

func (as *Arrays) NewITable() *ITable {
	return &ITable{}
}

func (t *ITable) AddCol(col string, v interface{}) *ITable {
	row := ensureRow(t)
	(*t)[row][col] = v
	return t
}

func (t *ITable) AddEmptyRow() *ITable {
	*t = append(*t, SIMap{})
	return t
}

func ensureRow(t *ITable) int {
	if len(*t) == 0 {
		*t = append(*t, SIMap{})
	}
	return len(*t) - 1
}
