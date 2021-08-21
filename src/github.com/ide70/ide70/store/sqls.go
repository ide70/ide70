package store

import (
	"encoding/json"
	"fmt"
	"github.com/ide70/ide70/loader"
	"strings"
)

var modSql *loader.TemplatedYaml

func loadDefs() {
	modSql = loader.LoadTemplatedYaml("postgresql/sqls", "ide70/db/")
}

func (dbCtx *DatabaseContext) SqlOP(name string) {

}

func (dbCtx *DatabaseContext) CRUDGenInsert(tableName string, data string) int64 {
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

func (dbCtx *DatabaseContext) CRUDGenFind(tableName string, id int64) map[string]interface{} {
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

func (dbCtx *DatabaseContext) CRUDGenUpdate(tableName string, id int64, data string) error {
	db := dbCtx.getConnection()
	defer db.Close()

	sql := "update "+tableName+" set data = $2 where id = $1;"

	_, err := db.Exec(sql, id, data)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func (dbCtx *DatabaseContext) CRUDGenDelete(tableName string, id int64) error {
	db := dbCtx.getConnection()
	defer db.Close()

	sql := "delete from "+tableName+" where id = $1;"

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

type ColumnOrder struct {
	fieldName string
	isAsc     bool
}

//func (dbCtx *DatabaseContext) WorksheetFindItemsPage(tableName string, allFilters map[string]*FilterTag, offset, pageSize int, orders []*ColumnOrder) []interface{} {
func (dbCtx *DatabaseContext) WorksheetFindItemsPage(tableName string, offset, pageSize int) []interface{} {
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

	datas := []interface{}{}
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
