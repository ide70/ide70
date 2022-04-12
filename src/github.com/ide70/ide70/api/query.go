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

type QueryCtx struct {
}

func (dbCtx *DatabaseContext) QueryCtx() *QueryCtx {
	return &QueryCtx{}
}

type ColumnOrder struct {
	column SchemaCol
	isAsc  bool
}

type QueryDef struct {
	from            *SchemaTableReference
	selectedColumns []SchemaCol
	condition       *QueryConditionWrapper
	ordering        []ColumnOrder
	offset          int
	limit           int
}

type SchemaTable map[string]SchemaCol

type SchemaTableReference struct {
	tableName string
	alias     string
}

type SchemaCol struct {
	tableRef *SchemaTableReference
	name     string
}

func (col SchemaCol) toSQL() string {
	if col.tableRef.alias != "" {
		return col.tableRef.alias + "." + col.name
	}
	return col.tableRef.tableName + "." + col.name
}

type QueryConditionWrapper struct {
	condition QueryCondition
}

type QueryCondition interface {
	toSQL() string
}

type Like struct {
	schemaCol SchemaCol
	likeExpr  string
}

func (like Like) toSQL() string {
	return like.schemaCol.toSQL() + " like " + sqlStringConst(like.likeExpr)
}

type Equals struct {
	schemaCol SchemaCol
	right     interface{}
}

func (equals Equals) toSQL() string {
	return equals.schemaCol.toSQL() + " = " + schemaColOrConstToSQL(equals.right)
}

func schemaColOrConstToSQL(i interface{}) string {
	switch it := i.(type) {
	case int, int64:
		return fmt.Sprintf("%d", i)
	case float32, float64:
		return fmt.Sprintf("%f", i)
	case string:
		return sqlStringConst(it)
	case time.Time:
		return "TIMESTAMP " + sqlStringConst(it.Format("2006-01-02 15:04:05"))
	case SchemaCol:
		return it.toSQL()
	}
	return "null"
}

type And struct {
	c1 QueryCondition
	c2 QueryCondition
}

func (and And) toSQL() string {
	return and.c1.toSQL() + " and " + and.c2.toSQL()
}

func (qc *QueryCtx) Table(tableName string) SchemaTable {
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
		table[columnName] = SchemaCol{name: columnName, tableRef: ref}
	}
	table[schemaTableReferenceKey] = SchemaCol{tableRef: ref}
	return table
}

func (qc *QueryCtx) From(table SchemaTable) *QueryDef {
	qd := &QueryDef{}
	qd.from = table[schemaTableReferenceKey].tableRef
	return qd
}

func (qd *QueryDef) Select(columns ...SchemaCol) *QueryDef {
	qd.selectedColumns = append(qd.selectedColumns, columns...)
	return qd
}

func (qd *QueryDef) Where(condition *QueryConditionWrapper) *QueryDef {
	qd.condition = condition
	return qd
}

func (qd *QueryDef) AscendingBy(column SchemaCol) *QueryDef {
	qd.ordering = append(qd.ordering, ColumnOrder{column: column, isAsc: true})
	return qd
}

func (qd *QueryDef) DescendingBy(column SchemaCol) *QueryDef {
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
	logger.Info("sql:", toSql(qd))
	return nil
}

func toSql(qd *QueryDef) string {
	qd.from.generateAlias(1)
	var sb strings.Builder
	sb.WriteString("select ")
	for idx, selectedColumn := range qd.selectedColumns {
		if idx > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(selectedColumn.toSQL())
	}
	sb.WriteString(" from ")
	sb.WriteString(qd.from.toSQL())
	if qd.condition != nil {
		sb.WriteString(" where ")
		sb.WriteString(qd.condition.condition.toSQL())
	}
	if len(qd.ordering) > 0 {
		for idx, orderColumn := range qd.ordering {
		if idx > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(orderColumn.toSQL())
	}
	}
	return sb.String()
}

func (schemaCol SchemaCol) Like(likeExpr string) *QueryConditionWrapper {
	like := Like{schemaCol: schemaCol, likeExpr: likeExpr}
	return &QueryConditionWrapper{condition: like}
}

func (schemaCol SchemaCol) Equals(likeExpr string) *QueryConditionWrapper {
	like := Like{schemaCol: schemaCol, likeExpr: likeExpr}
	return &QueryConditionWrapper{condition: like}
}

func (c *QueryConditionWrapper) And(cRight *QueryConditionWrapper) *QueryConditionWrapper {
	and := And{c1: c.condition, c2: cRight.condition}
	return &QueryConditionWrapper{condition: and}
}

func sqlStringConst(s string) string {
	return "'" + s + "'"
}

func (str *SchemaTableReference) generateAlias(idx int) {
	str.alias = fmt.Sprintf("T%d", idx)
}

func (str *SchemaTableReference) toSQL() string {
	sql := str.tableName
	if str.alias != "" {
		sql += " " + str.alias
	}
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
