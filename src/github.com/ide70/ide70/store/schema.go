package store

import (
	"github.com/ide70/ide70/dataxform"
)

type TableInfo struct {
	Valid bool
}

type SchemaInfo struct {
	currentSchema string
	Tables        map[string]*TableInfo
}

func initSchemaInfo(dbCtx *DatabaseContext) {
	schemaInfo := &SchemaInfo{Tables: map[string]*TableInfo{}}
	schemaInfo.currentSchema = dataxform.IAsString(dbCtx.SQLGetValue("select current_schema;"))
	dbCtx.SchemaInfo = schemaInfo
}

func ensureTable(dbCtx *DatabaseContext, tableName string) {
	if dbCtx.SchemaInfo == nil {
		logger.Info("initSchemaInfo")
		initSchemaInfo(dbCtx)
	}
	if dbCtx.SchemaInfo.Tables[tableName] == nil {
		logger.Info("no table", tableName, "exists in schemaInfo")
		tableExists := dataxform.IAsBool(dbCtx.SQLGetValue(
			`SELECT EXISTS (
   SELECT FROM information_schema.tables 
   WHERE  table_schema = $1
   AND    table_name   = $2
   );`, dbCtx.SchemaInfo.currentSchema, tableName))
		
		if !tableExists {
			logger.Info("Creating table:", tableName)
			// read configuration
			// related tables, etc.
			createTable(dbCtx, tableName)
		}
		tableInfo := &TableInfo{Valid: true}
		dbCtx.SchemaInfo.Tables[tableName] = tableInfo
	}
}

func createTable(dbCtx *DatabaseContext, tableName string) {
	dbCtx.SQLExec(`create table ` + tableName + ` (
	id bigserial primary key,
	data jsonb);`)
}
