package api

import (
	"github.com/ide70/ide70/loader"
)

/*

---
descr: Flight reservation
columns:
  - name: flight_id
  - name: seat_id
  - name: price
connections:
  flightConn:
    column: flight_id
    foreignTable: flight
  seatConn:
    column: seat_id
    foreignTable: seat

connections:
  flightMConn:
    column: _id
    foreignTable: flight
    foreignColumn: airplane_id

---
descr: Airplane ticket
columns:
  - name: ticket_code
connections:
  reservationConn:
    foreignTable: flightReservation
    mx: ticket_reservation

*/

type TableStruct struct {
	name string
	tableType string
	columns     map[string]*TableColumn
	connections map[string]*TableConnection
}

type TableConnection struct {
	foreignTable  *TableStruct
	foreignColumn *TableColumn
	column        *TableColumn
	mxTableName   string
}

type TableColumn struct {
	name    string
	idField bool
}

var tableStructCache = map[string]*TableStruct{}

func getTableStruct(tableName string) *TableStruct {
	if tableStruct, has := tableStructCache[tableName]; has {
		return tableStruct
	}
	return loadTableStruct(tableName)
}

func loadTableStruct(tableName string) *TableStruct {
	table := &TableStruct{name: tableName, columns: map[string]*TableColumn{}, connections: map[string]*TableConnection{}}

	fileAsTemplatedYaml := loader.GetTemplatedYaml(tableName, "ide70/dcfg/schema/")
	if fileAsTemplatedYaml == nil {
		table.tableType = TABLETYPE_JSONB
		tableStructCache[tableName] = table
		return table
	}
	tableType := SIMapGetByKeyAsString(fileAsTemplatedYaml.Def, "type")
	if tableType == "" {
		tableType = TABLETYPE_JSONB
	}
	table.tableType = tableType
	columnList := SIMapGetByKeyAsList(fileAsTemplatedYaml.Def, "columns")
	for _, columnIf := range columnList {
		column := IAsSIMap(columnIf)
		columnName := SIMapGetByKeyAsString(column, "name")
		table.columns[columnName] = &TableColumn{name: columnName, idField: false}
	}
	table.columns[idFieldName] = &TableColumn{name: "id", idField: true}
	tableStructCache[tableName] = table

	// load connections

	connMap := SIMapGetByKeyAsMap(fileAsTemplatedYaml.Def, "connections")
	logger.Debug("connMap:", connMap)
	for connName, connIf := range connMap {
		logger.Debug("connName:", connName)
		tableConn := &TableConnection{}
		conn := IAsSIMap(connIf)
		logger.Debug("conn:", conn)
		localColumnName := SIMapGetByKeyAsString(conn, "column")
		foreignTableName := SIMapGetByKeyAsString(conn, "foreignTable")
		foreignColumnName := SIMapGetByKeyAsString(conn, "foreignColumn")
		mxTableName := SIMapGetByKeyAsString(conn, "mx")
		if foreignColumnName == "" {
			foreignColumnName = idFieldName
		}
		if localColumnName == "" {
			localColumnName = idFieldName
		}
		tableConn.column = table.columns[localColumnName]
		tableConn.foreignTable = getTableStruct(foreignTableName)
		tableConn.foreignColumn = tableConn.foreignTable.columns[foreignColumnName]
		tableConn.mxTableName = mxTableName
		table.connections[connName] = tableConn
	}

	return table
}
