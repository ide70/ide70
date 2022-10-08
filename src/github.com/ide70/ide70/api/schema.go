package api

type TableInfo struct {
	Valid bool
}

type SchemaInfo struct {
	currentSchema string
	Tables        map[string]*TableInfo
}

func initSchemaInfo(dbCtx *DatabaseContext) {
	schemaInfo := &SchemaInfo{Tables: map[string]*TableInfo{}}
	schemaInfo.currentSchema = IAsString(dbCtx.SQLGetValue("select current_schema;"))
	dbCtx.SchemaInfo = schemaInfo
}

func ensureTable(dbCtx *DatabaseContext, tableName string, tableType string) {
	if dbCtx.SchemaInfo == nil {
		logger.Debug("initSchemaInfo")
		initSchemaInfo(dbCtx)
	}
	if dbCtx.SchemaInfo.Tables[tableName] == nil {
		logger.Debug("no table", tableName, "exists in schemaInfo")
		tableExists := IAsBool(dbCtx.SQLGetValue(
			`SELECT EXISTS (
   SELECT FROM information_schema.tables 
   WHERE  table_schema = $1
   AND    table_name   = $2
   );`, dbCtx.SchemaInfo.currentSchema, tableName))

		if !tableExists {
			logger.Debug("Creating table:", tableName)
			// read configuration
			// related tables, etc.
			createTable(dbCtx, tableName, tableType)
		}
		tableInfo := &TableInfo{Valid: true}
		dbCtx.SchemaInfo.Tables[tableName] = tableInfo
	}
}

func createTable(dbCtx *DatabaseContext, tableName string, tableType string) {
	switch tableType {
	case TABLETYPE_JSONB:
		dbCtx.SQLExec(`create table ` + tableName + ` (
	id bigserial primary key,
	data jsonb);`)
	case TABLETYPE_BLOB:
		dbCtx.SQLExec(`create table ` + tableName + ` (
	id bigserial primary key,
	data bytea);`)
	}

}
