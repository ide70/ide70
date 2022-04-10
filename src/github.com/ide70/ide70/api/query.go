package api

import (
	"github.com/ide70/ide70/loader"
    "github.com/ide70/ide70/dataxform"
    "time"
    "fmt"
//	"regexp"
//	"strings"
)

type QueryCtx struct {
	
}

type SchemaTable map[string]SchemaCol

type SchemaTableReference struct {
	tableName string
}

type SchemaCol struct {
	tableRef *SchemaTableReference
	name string
}

func (col SchemaCol) toSQL() string {
	return col.tableRef.tableName + "." + col.name
}

type QueryConditionWrapper struct {
	condition QueryCondition
}

type QueryCondition interface{
	toSQL() string
}

type Like struct {
	schemaCol SchemaCol
	likeExpr string
}

func (like Like) toSQL() string {
	return like.schemaCol.toSQL() + " like " + sqlStringConst(like.likeExpr)
}

type Equals struct {
	schemaCol SchemaCol
	right interface{}
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
		case  SchemaCol:
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
	for _,columnIf := range columnList {
		column := dataxform.IAsSIMap(columnIf)
		columnName := dataxform.SIMapGetByKeyAsString(column, "name")
		table[columnName] = SchemaCol{name:columnName,tableRef:ref}
	}
	return table
}

func (schemaCol SchemaCol) Like(likeExpr string) QueryConditionWrapper {
	like := Like{schemaCol: schemaCol, likeExpr: likeExpr}
	return QueryConditionWrapper{condition: like}
}

func (schemaCol SchemaCol) Equals(likeExpr string) QueryConditionWrapper {
	like := Like{schemaCol: schemaCol, likeExpr: likeExpr}
	return QueryConditionWrapper{condition: like}
}

func (c QueryConditionWrapper) And(cRight QueryConditionWrapper) QueryConditionWrapper {
	and := And{c1:c.condition,c2:cRight.condition}
	return QueryConditionWrapper{condition: and}
}

func sqlStringConst(s string) string {
	return "'" +s + "'"
}
