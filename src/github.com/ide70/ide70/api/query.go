package api

import (
	"fmt"
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
	"time"
	//	"regexp"
	"strings"
)

const schemaTableReferenceKey = "__tableReference"
const idFieldName = "_id"
const dataFieldName = "data"

type QueryCtx struct {
	dbCtx *DatabaseContext
}

func (dbCtx *DatabaseContext) QueryCtx() *QueryCtx {
	return &QueryCtx{dbCtx: dbCtx}
}

type ColumnOrder struct {
	column *SchemaCol
	isAsc  bool
}

type QueryDef struct {
	qc              *QueryCtx
	from            *SchemaTableReference
	connections     map[string]*SchemaTableReference
	selectedColumns []*SchemaCol
	condition       *QueryConditionWrapper
	ordering        []ColumnOrder
	offset          int
	limit           int
}

type SchemaTable map[string]*SchemaCol

type SchemaTableReference struct {
	tableName        string
	alias            string
	parentConnection *SchemaConnection
}

type SchemaConnection struct {
	parentTableRef *SchemaTableReference
	joinCondition  QueryCondition
	uniqueId       string
}

type SchemaCol struct {
	tableRef *SchemaTableReference
	name     string
	idField  bool
}

func (col *SchemaCol) toSQL() string {
	return col.toSQLWithConversion("")
}

func (col *SchemaCol) toSQLWithConversion(dataTypeConv string) string {
	if col.tableRef.alias != "" {
		return col.columnSQL(col.tableRef.alias, dataTypeConv)
	}
	return col.columnSQL(col.tableRef.tableName, dataTypeConv)
}

func (col *SchemaCol) columnSQL(tableName, dataTypeConv string) string {
	if col.idField {
		return tableName + "." + col.name
	}
	if dataTypeConv != "" {
		return "(" + tableName + "." + dataFieldName + "->'" + col.name + "')::" + dataTypeConv
	}
	return tableName + "." + dataFieldName + "->>'" + col.name + "'"
}

type QueryConditionWrapper struct {
	condition        QueryCondition
	conditionColumns []*SchemaCol
}

type QueryCondition interface {
	toSQL() string
}

type Like struct {
	schemaCol *SchemaCol
	likeExpr  string
}

func (like Like) toSQL() string {
	return like.schemaCol.toSQL() + " like " + sqlStringConst(like.likeExpr)
}

type Equals struct {
	schemaCol *SchemaCol
	right     interface{}
}

func (equals Equals) toSQL() string {
	if equals.right == nil {
		return "1 = 1"
	}
	dc1, dc2 := autoSQLDataTypeConversion(equals.schemaCol, equals.right)
	return equals.schemaCol.toSQLWithConversion(dc1) + " = " + schemaColOrConstToSQL(equals.right, dc2)
}

func autoSQLDataTypeConversion(col1, col2 interface{}) (string, string) {
	switch col1T := col1.(type) {
	case *SchemaCol:
		switch col2T := col2.(type) {
		case *SchemaCol:
			if col1T.idField && !col2T.idField {
				return "", "numeric"
			}
			if !col1T.idField && col2T.idField {
				return "numeric", ""
			}
		}

	}
	return "", ""
}

func schemaColOrConstToSQL(i interface{}, dataTypeConversion string) string {
	switch it := i.(type) {
	case int, int64:
		return fmt.Sprintf("%d", i)
	case float32, float64:
		return fmt.Sprintf("%f", i)
	case string:
		return sqlStringConst(it)
	case time.Time:
		return "TIMESTAMP " + sqlStringConst(it.Format("2006-01-02 15:04:05"))
	case *SchemaCol:
		return it.toSQLWithConversion(dataTypeConversion)
	}
	return "null"
}

func isNumeric(i interface{}) bool {
	switch it := i.(type) {
	case int, int64:
		return true
	case float32, float64:
		return true
	case string:
		return false
	case time.Time:
		return false
	case *SchemaCol:
		return it.idField
	}
	return false
}

type And struct {
	c1 QueryCondition
	c2 QueryCondition
}

type Or struct {
	c1 QueryCondition
	c2 QueryCondition
}

type TrueCondition struct {
}

func (or Or) toSQL() string {
	return or.c1.toSQL() + " or " + or.c2.toSQL()
}

func (and And) toSQL() string {
	return and.c1.toSQL() + " and " + and.c2.toSQL()
}

func (ec TrueCondition) toSQL() string {
	return ""
}

func (qc *QueryCtx) Table(tableName string) SchemaTable {
	return newSchemaTable(tableName)
}

func newSchemaTable(tableName string) SchemaTable {
	table := SchemaTable{}
	ref := &SchemaTableReference{tableName: tableName}
	fileAsTemplatedYaml := loader.GetTemplatedYaml(tableName, "ide70/dcfg/schema/")
	if fileAsTemplatedYaml == nil {
		return table
	}
	columnList := dataxform.SIMapGetByKeyAsList(fileAsTemplatedYaml.Def, "columns")
	for _, columnIf := range columnList {
		column := dataxform.IAsSIMap(columnIf)
		columnName := dataxform.SIMapGetByKeyAsString(column, "name")
		table[columnName] = &SchemaCol{name: columnName, tableRef: ref, idField: false}
	}
	table[idFieldName] = &SchemaCol{name: "id", tableRef: ref, idField: true}
	table[schemaTableReferenceKey] = &SchemaCol{tableRef: ref}
	return table
}

func (st SchemaTable) JoinedTable(connectionName string) SchemaTable {
	table := SchemaTable{}
	parentRef := st[schemaTableReferenceKey].tableRef
	parentTableName := parentRef.tableName
	fileAsTemplatedYaml := loader.GetTemplatedYaml(parentTableName, "ide70/dcfg/schema/")
	if fileAsTemplatedYaml == nil {
		return table
	}
	connMap := dataxform.SIMapGetByKeyAsMap(fileAsTemplatedYaml.Def, "connections")
	logger.Info("connMap:", connMap)
	for connName, connIf := range connMap {
		if connName != connectionName {
			continue
		}
		logger.Info("connName:", connName)
		conn := dataxform.IAsSIMap(connIf)
		logger.Info("conn:", conn)
		localColumnName := dataxform.SIMapGetByKeyAsString(conn, "column")
		foreignTableName := dataxform.SIMapGetByKeyAsString(conn, "foreignTable")
		foreignColumnName := dataxform.SIMapGetByKeyAsString(conn, "foreignColumn")
		if foreignColumnName == "" {
			foreignColumnName = idFieldName
		}
		table = newSchemaTable(foreignTableName)
		ref := table[schemaTableReferenceKey].tableRef
		parentUniqueId := parentTableName
		if parentRef.parentConnection != nil {
			parentUniqueId = parentRef.parentConnection.uniqueId
		}
		condition := Equals{schemaCol: st[localColumnName], right: table[foreignColumnName]}
		ref.parentConnection = &SchemaConnection{parentTableRef: parentRef, joinCondition: condition, uniqueId: parentUniqueId + "." + connectionName}
		break
	}
	logger.Info("JoinedTable:", table)

	return table
}

func newQueryDef() *QueryDef {
	return &QueryDef{connections: map[string]*SchemaTableReference{}}
}

func (qc *QueryCtx) From(table SchemaTable) *QueryDef {
	qd := newQueryDef()
	qd.qc = qc
	qd.from = table[schemaTableReferenceKey].tableRef
	return qd
}

func (qc *QueryCtx) NewQuery() *QueryDef {
	qd := newQueryDef()
	qd.qc = qc
	return qd
}

func (qd *QueryDef) From(table SchemaTable) *QueryDef {
	qd.from = table[schemaTableReferenceKey].tableRef
	return qd
}

func (qc *QueryCtx) Join(table SchemaTable) *QueryDef {
	qd := newQueryDef()
	qd.qc = qc
	qd.from = table[schemaTableReferenceKey].tableRef
	return qd
}

func (qd *QueryDef) Select(columns ...*SchemaCol) *QueryDef {
	qd.selectedColumns = append(qd.selectedColumns, columns...)
	return qd
}

func (qd *QueryDef) Where(condition *QueryConditionWrapper) *QueryDef {
	qd.condition = condition
	return qd
}

func (qd *QueryDef) AscendingBy(column *SchemaCol) *QueryDef {
	qd.ordering = append(qd.ordering, ColumnOrder{column: column, isAsc: true})
	return qd
}

func (qd *QueryDef) DescendingBy(column *SchemaCol) *QueryDef {
	qd.ordering = append(qd.ordering, ColumnOrder{column: column, isAsc: false})
	return qd
}

func (qd *QueryDef) Limit(limitIf interface{}) *QueryDef {
	limit := dataxform.IAsInt(limitIf)
	qd.limit = limit
	return qd
}

func (qd *QueryDef) Offset(offsetIf interface{}) *QueryDef {
	offset := dataxform.IAsInt(offsetIf)
	qd.offset = offset
	return qd
}

func (qd *QueryDef) List() ITable {
	return qd.qc.dbCtx.RunQueryDef(qd)
}

func (qd *QueryDef) OneRow() SIMap {
	qd.Limit(1)
	resultTable := qd.qc.dbCtx.RunQueryDef(qd)
	if len(resultTable) == 0 {
		return nil
	}
	return resultTable[0]
}

func (qd *QueryDef) _toSQL() string {
	aliasIdx := 1
	qd.from.generateAlias(aliasIdx)
	aliasIdx++
	logger.Info("connections generation start")
	qd.lookupConnections()
	logger.Info("connections generated")
	for _, conn := range qd.connections {
		conn.generateAlias(aliasIdx)
		aliasIdx++
	}
	logger.Info("aliases for connections generated")
	var sb strings.Builder
	sb.WriteString("select ")
	for idx, selectedColumn := range qd.selectedColumns {
		if idx > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(selectedColumn.toSQL())
	}
	sb.WriteString(" from ")
	sb.WriteString(qd.from.toTableSQL())

	// generate joins
	for _, join := range qd.connections {
		logger.Info("generating join:", join)
		sb.WriteString(join.toJoinSQL())
	}
	logger.Info("joins generated")

	if qd.condition != nil {
		sb.WriteString(" where ")
		sb.WriteString(qd.condition.condition.toSQL())
	}
	if len(qd.ordering) > 0 {
		sb.WriteString(" order by ")
		for idx, orderColumn := range qd.ordering {
			if idx > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(orderColumn.toSQL())
		}
	}
	if qd.offset != 0 {
		sb.WriteString(" offset ")
		sb.WriteString(fmt.Sprintf("%d", qd.offset))
	}
	if qd.limit != 0 {
		sb.WriteString(" limit ")
		sb.WriteString(fmt.Sprintf("%d", qd.limit))
	}
	return sb.String()
}

func (qd *QueryDef) lookupConnections() {
	logger.Info("qd.selectedColumns", qd.selectedColumns)
	for _, selectedColumn := range qd.selectedColumns {
		qd.addConnectingTable(selectedColumn.tableRef)
	}
	if qd.condition != nil {
		for _, conditionColumn := range qd.condition.conditionColumns {
			qd.addConnectingTable(conditionColumn.tableRef)
		}
	}
}

func (qd *QueryDef) addConnectingTable(tableRef *SchemaTableReference) {
	logger.Info("addConnectingTable tableRef:", tableRef)
	conn := tableRef.parentConnection
	if conn != nil {
		logger.Info("conn:", conn)
		if qd.connections[conn.uniqueId] == nil {
			qd.addConnectingTable(conn.parentTableRef)
			qd.connections[conn.uniqueId] = tableRef
		}
	}
}

func (schemaCol *SchemaCol) Like(likeExpr string) *QueryConditionWrapper {
	like := Like{schemaCol: schemaCol, likeExpr: likeExpr}
	return &QueryConditionWrapper{condition: like, conditionColumns: []*SchemaCol{schemaCol}}
}

func (schemaCol *SchemaCol) Equals(right interface{}) *QueryConditionWrapper {
	like := Equals{schemaCol: schemaCol, right: right}
	qcw := &QueryConditionWrapper{condition: like}
	conditionColumns := []*SchemaCol{schemaCol}
	switch rightT := right.(type) {
	case *SchemaCol:
		conditionColumns = append(conditionColumns, rightT)
	}
	qcw.conditionColumns = conditionColumns
	return qcw
}

func (c *QueryConditionWrapper) OrEmpty(right interface{}) *QueryConditionWrapper {
	empty := dataxform.IsEmpty(right)
	if empty {
		return &QueryConditionWrapper{condition: TrueCondition{}, conditionColumns: c.conditionColumns}
	} else {
		return c
	}
}

func isTrueCondition(c *QueryConditionWrapper) bool {
	switch c.condition.(type) {
		case TrueCondition:
		return true
	}
	return false
}

func (c *QueryConditionWrapper) And(cRight *QueryConditionWrapper) *QueryConditionWrapper {
	if isTrueCondition(cRight) {
		return &QueryConditionWrapper{condition: c.condition, conditionColumns: append(c.conditionColumns, cRight.conditionColumns...)}
	}
	if isTrueCondition(c) {
		return &QueryConditionWrapper{condition: cRight.condition, conditionColumns: append(c.conditionColumns, cRight.conditionColumns...)}
	}
	and := And{c1: c.condition, c2: cRight.condition}
	return &QueryConditionWrapper{condition: and, conditionColumns: append(c.conditionColumns, cRight.conditionColumns...)}
}

func (c *QueryConditionWrapper) Or(cRight *QueryConditionWrapper) *QueryConditionWrapper {
	if isTrueCondition(cRight) {
		return &QueryConditionWrapper{condition: TrueCondition{}, conditionColumns: append(c.conditionColumns, cRight.conditionColumns...)}
	}
	if isTrueCondition(c) {
		return &QueryConditionWrapper{condition: TrueCondition{}, conditionColumns: append(c.conditionColumns, cRight.conditionColumns...)}
	}
	or := Or{c1: c.condition, c2: cRight.condition}
	return &QueryConditionWrapper{condition: or, conditionColumns: append(c.conditionColumns, cRight.conditionColumns...)}
}

func sqlStringConst(s string) string {
	return "'" + s + "'"
}

func (str *SchemaTableReference) generateAlias(idx int) {
	str.alias = fmt.Sprintf("T%d", idx)
}

func (str *SchemaTableReference) toTableSQL() string {
	sql := str.tableName
	if str.alias != "" {
		sql += " " + str.alias
	}
	return sql
}

func (str *SchemaTableReference) toJoinSQL() string {
	sql := " JOIN "
	sql += str.toTableSQL()
	logger.Info("str.parentConnection:", str.parentConnection)
	sql += " ON "
	sql += str.parentConnection.joinCondition.toSQL()
	return sql
}

func (corder ColumnOrder) toSQL() string {
	sql := corder.column.toSQL() + " "
	if corder.isAsc {
		sql += "asc"
	} else {
		sql += "desc"
	}
	return sql
}
