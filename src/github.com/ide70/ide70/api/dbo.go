package api

import (
	"github.com/ide70/ide70/dataxform"
)

type DataBaseObjectKey struct {
	Value int64
}

type DataBaseObject struct {
	Data      map[string]interface{}
	TableName string
	Key       *DataBaseObjectKey
	dbCtx     *DatabaseContext
	// TODO: changed flag
}

func (dbCtx *DatabaseContext) CreateDBO(data map[string]interface{}, tableName string) *DataBaseObject {
	dbo := &DataBaseObject{dbCtx: dbCtx, Data: data, TableName: tableName}
	return dbo
}

func (dbCtx *DatabaseContext) FindDBO(tableName string, key int64) *DataBaseObject {
	data := dbCtx.CRUDGenFind(tableName, key)
	if data == nil {
		return nil
	}
	dbo := &DataBaseObject{dbCtx: dbCtx, Data: data, TableName: tableName, Key: &DataBaseObjectKey{Value: key}}
	return dbo
}

func (dbo *DataBaseObject) GetData() SIMap{
	return dbo.Data
}

func (dbo *DataBaseObject) UpdateData(data map[string]interface{}) {
	dbo.Data = data
	// TODO: - detect real changes
}

func (dbo *DataBaseObject) Save() {
	if dbo.Key == nil {
		// new dbo
		logger.Info("save dbo")
		keyValue := dbo.dbCtx.CRUDGenInsert(dbo.TableName, dataxform.SIMapToJson(dbo.Data))
		dbo.Key = &DataBaseObjectKey{Value: keyValue}
	} else {
		// existing dbo
		dbo.dbCtx.CRUDGenUpdate(dbo.TableName, dbo.Key.Value, dataxform.SIMapToJson(dbo.Data))
	}
}

func (dbo *DataBaseObject) Delete() {
	if dbo.Key != nil {
		logger.Info("delete dbo")
		dbo.dbCtx.CRUDGenDelete(dbo.TableName, dbo.Key.Value)
	}
}
