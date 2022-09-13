package api

import (
	"encoding/json"
	"fmt"
	"github.com/ide70/ide70/loader"
	"strings"
)

const (
	TABLETYPE_JSONB = "jsonb"
	TABLETYPE_BLOB = "blob"
)


var modSql *loader.TemplatedYaml

func loadDefs() {
	modSql = loader.LoadTemplatedYaml("postgresql/sqls", "ide70/db/")
}

func (dbCtx *DatabaseContext) SqlOP(name string) {

}

func (dbCtx *DatabaseContext) SQLGetValue(sql string, sqlParams ...interface{}) interface{} {
	db := dbCtx.getConnection()
	defer db.Close()

	logger.Info("sql: ", sql)
	rows, err := db.Query(sql, sqlParams...)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	if rows.Next() {
		var data interface{}

		err := rows.Scan(&data)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		return data
	}
	logger.Warning("value not found")

	return nil
}

func (dbCtx *DatabaseContext) CRUDGenInsert(tableName string, data string) int64 {
	ensureTable(dbCtx, tableName, TABLETYPE_JSONB)
	db := dbCtx.getConnection()
	defer db.Close()

	sql := "insert into " + tableName + " (data) values($1) returning id;"
	logger.Info("sql:", sql)
	var id int64
	err := db.QueryRow(sql, data).Scan(&id)
	if err != nil {
		fmt.Println(err)
	}

	return id
}

func (dbCtx *DatabaseContext) CRUDGenInsertBlob(tableName string, data *BinaryData) int64 {
	ensureTable(dbCtx, tableName, TABLETYPE_BLOB)
	db := dbCtx.getConnection()
	defer db.Close()

	sql := "insert into " + tableName + " (data) values($1) returning id;"
	logger.Info("sql:", sql)
	var id int64
	err := db.QueryRow(sql, *data.data).Scan(&id)
	if err != nil {
		fmt.Println(err)
	}

	return id
}

//select data from mx_ticket_reservation where (data->'id_ticket')::numeric = 24 and (data->'ord')::numeric = 0;

func criteriasToSql(criterias...*ColumnCriteria) string{
	var sb strings.Builder
	for idx,criteria := range criterias {
		if idx>0 {
			sb.WriteString(" AND ")
		}
		sb.WriteString(fmt.Sprintf("(data->'%s')::numeric = $%d", criteria.column, idx+1))
	}
	return sb.String()
}

func criteriasToArgs(criterias...*ColumnCriteria) []interface{}{
	args := []interface{}{}
	for _,criteria := range criterias {
		args = append(args, criteria.value)
	}
	return args
}

func (dbCtx *DatabaseContext) CRUDGenFindbyCriteria(tableName string, criterias...*ColumnCriteria) map[string]interface{} {
	ensureTable(dbCtx, tableName, TABLETYPE_JSONB)
	db := dbCtx.getConnection()
	defer db.Close()

	sql := "select id, data from " + tableName + " where "+criteriasToSql(criterias...)+";"
	logger.Info("sql:", sql)
	args := criteriasToArgs(criterias...)
	logger.Info("args:", args)

	rows, err := db.Query(sql, args...)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	if rows.Next() {
		var id int64
		var data string

		err := rows.Scan(&id, &data)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		dataSet := map[string]interface{}{}
		decoder := json.NewDecoder(strings.NewReader(data))
		decoder.Decode(&dataSet)

		dataSet["_id"] = id

		return dataSet
	}

	return nil
}

func (dbCtx *DatabaseContext) CRUDGenFind(tableName string, id int64) map[string]interface{} {
	ensureTable(dbCtx, tableName, TABLETYPE_JSONB)
	db := dbCtx.getConnection()
	defer db.Close()

	sql := "select data from " + tableName + " where id = $1;"

	rows, err := db.Query(sql, id)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	if rows.Next() {
		var data string

		err := rows.Scan(&data)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		dataSet := map[string]interface{}{}
		decoder := json.NewDecoder(strings.NewReader(data))
		decoder.Decode(&dataSet)

		dataSet["_id"] = id

		return dataSet
	}

	return nil
}

func (dbCtx *DatabaseContext) GenLoadBlob(tableName string, id int64) *[]byte {
	ensureTable(dbCtx, tableName, TABLETYPE_BLOB)
	db := dbCtx.getConnection()
	defer db.Close()

	sql := "select data from " + tableName + " where id = $1;"

	rows, err := db.Query(sql, id)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	if rows.Next() {
		var data []byte

		err := rows.Scan(&data)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		return &data
	}

	return nil
}

func (dbCtx *DatabaseContext) CRUDGenUpdate(tableName string, id int64, data string) error {
	ensureTable(dbCtx, tableName, TABLETYPE_JSONB)
	db := dbCtx.getConnection()
	defer db.Close()

	sql := "update " + tableName + " set data = $2 where id = $1;"

	_, err := db.Exec(sql, id, data)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func (dbCtx *DatabaseContext) CRUDGenUpdateBlob(tableName string, id int64, data *BinaryData) error {
	ensureTable(dbCtx, tableName, TABLETYPE_BLOB)
	db := dbCtx.getConnection()
	defer db.Close()

	sql := "update " + tableName + " set data = $2 where id = $1;"

	_, err := db.Exec(sql, id, *data.data)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func (dbCtx *DatabaseContext) SQLExec(sql string, sqlParams ...interface{}) error {
	db := dbCtx.getConnection()
	defer db.Close()

	_, err := db.Exec(sql, sqlParams...)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func (dbCtx *DatabaseContext) CRUDGenDelete(tableName string, id int64) error {
	ensureTable(dbCtx, tableName, TABLETYPE_JSONB)
	db := dbCtx.getConnection()
	defer db.Close()

	sql := "delete from " + tableName + " where id = $1;"

	_, err := db.Exec(sql, id)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func (dbCtx *DatabaseContext) WorksheetFindKeys(key_type string) []string {
	db := dbCtx.getConnection()
	defer db.Close()

	sql := `
select key from worksheet_instance where key_type = $1;
`

	rows, err := db.Query(sql, key_type)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	keys := []string{}
	for rows.Next() {
		var key string

		err := rows.Scan(&key)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		keys = append(keys, key)
	}

	return keys
}

type FilterTag struct {
	Comp1     string
	Comp2     string
	Relation  string
	CompareAs string
}

type FilterGroup struct {
	FilterTags []*FilterTag
}

//func (dbCtx *DatabaseContext) WorksheetFindItemsPage(tableName string, allFilters map[string]*FilterTag, offset, pageSize int, orders []*ColumnOrder) []interface{} {
func (dbCtx *DatabaseContext) WorksheetFindItemsPage(tableName string, offset, pageSize int) ITable {
	ensureTable(dbCtx, tableName, TABLETYPE_JSONB)
	//	filters := getFilterGroups(allFilters)
	db := dbCtx.getConnection()
	defer db.Close()

	sql := "select id, data from " + tableName + " where 1=1 "
	/*	if filters != nil {
			if filtersSql := filters.extractSQL(); filtersSql != "" {
				sql += " and " + filters.extractSQL()
			}
		}

		sql += extractOrderSQL(orders)
	*/
	if pageSize > 0 {
		sql += fmt.Sprintf(" limit %d", pageSize)
	}
	if offset > 0 {
		sql += fmt.Sprintf(" offset %d", offset)
	}
	sql += ";"
	fmt.Println("sql:", sql)

	rows, err := db.Query(sql)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	datas := ITable{}
	for rows.Next() {
		var id int64
		var dataStr string

		err := rows.Scan(&id, &dataStr)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		dataSet := map[string]interface{}{}
		decoder := json.NewDecoder(strings.NewReader(dataStr))
		decoder.Decode(&dataSet)

		dataSet["_id"] = id

		datas = append(datas, dataSet)
	}

	return datas
}

func (dbCtx *DatabaseContext) RunQueryDef(qd *QueryDef) ITable {
	ensureTable(dbCtx, qd.from.tableName, TABLETYPE_JSONB)
	
	db := dbCtx.getConnection()
	defer db.Close()

	sql := qd._toSQL()
	sql += ";"
	fmt.Println("sql:", sql)

	rows, err := db.Query(sql)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	datas := ITable{}
	for rows.Next() {
		colsPtr := []interface{}{}
		for i := 0; i < len(qd.selectedColumns); i++ {
			var colData interface{}
			colsPtr = append(colsPtr, &colData)
		}
		

		err := rows.Scan(colsPtr...)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		dataSet := map[string]interface{}{}
		
		for idx,selectedColumn := range qd.selectedColumns {
			// TODO: column alias handling
			dataSet[selectedColumn.name] = *colsPtr[idx].(*interface{})
		}

		datas = append(datas, dataSet)
	}

	return datas
}
