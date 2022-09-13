package api

type DataBaseObjectKey struct {
	Value int64
}

type ForeignKey struct {
	foreignDBO *DataBaseObject
	columnName string
}

type DataBaseObject struct {
	Data        map[string]interface{}
	BinaryData  *BinaryData
	TableName   string
	Key         *DataBaseObjectKey
	dbCtx       *DatabaseContext
	foreignKeys []*ForeignKey
	// TODO: changed flag
	saveOrder int
	toDelete  bool
}

func (dbCtx *DatabaseContext) CreateDBO(data map[string]interface{}, tableName string) *DataBaseObject {
	dbo := &DataBaseObject{dbCtx: dbCtx, Data: data, TableName: tableName}
	return dbo
}

func (dbCtx *DatabaseContext) CreateBinaryDBO(data *BinaryData, tableName string) *DataBaseObject {
	dbo := &DataBaseObject{dbCtx: dbCtx, BinaryData: data, TableName: tableName}
	return dbo
}

func (dbCtx *DatabaseContext) CreateNewDBO(tableName string) *DataBaseObject {
	dbo := &DataBaseObject{dbCtx: dbCtx, Data: map[string]interface{}{}, TableName: tableName}
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

type ColumnCriteria struct {
	column string
	value  int64
}

func (dbCtx *DatabaseContext) FindDBObyCriteria(tableName string, criterias ...*ColumnCriteria) *DataBaseObject {
	data := dbCtx.CRUDGenFindbyCriteria(tableName, criterias...)
	if data == nil {
		return nil
	}
	dbo := &DataBaseObject{dbCtx: dbCtx, Data: data, TableName: tableName, Key: &DataBaseObjectKey{Value: data["_id"].(int64)}}
	return dbo
}

func (dbo *DataBaseObject) GetData() SIMap {
	return dbo.Data
}

func (dbo *DataBaseObject) MarkToDelete() {
	dbo.toDelete = true
}

func (dbo *DataBaseObject) IsMarkedToDelete() bool {
	return dbo.toDelete
}

func (dbo *DataBaseObject) UpdateData(data map[string]interface{}) {
	dbo.Data = data
	// TODO: - detect real changes
}

func (dbo *DataBaseObject) UpdateBinaryData(data *BinaryData) {
	dbo.BinaryData = data
	// TODO: - detect real changes, by checksum
}

func (dbo *DataBaseObject) UpdateForeignKeys() {
	for _, foreignKey := range dbo.foreignKeys {
		logger.Info("updating f key:", foreignKey.columnName, "to:", foreignKey.foreignDBO.Key.Value)
		SIMapUpdateValue(foreignKey.columnName, foreignKey.foreignDBO.Key.Value, dbo.Data, true)
	}
}

func (dbo *DataBaseObject) addForeignKey(foreignKey *ForeignKey) {
	dbo.foreignKeys = append(dbo.foreignKeys, foreignKey)
}

func (dbo *DataBaseObject) Save() {
	if dbo.toDelete {
		return
	}
	if dbo.Key == nil {
		// new dbo
		logger.Info("save dbo")
		var keyValue int64
		if dbo.BinaryData != nil {
			keyValue = dbo.dbCtx.CRUDGenInsertBlob(dbo.TableName, dbo.BinaryData)
		} else {
			keyValue = dbo.dbCtx.CRUDGenInsert(dbo.TableName, SIMapToJson(dbo.Data))
		}
		dbo.Key = &DataBaseObjectKey{Value: keyValue}
	} else {
		// existing dbo
		if dbo.BinaryData != nil {
			dbo.dbCtx.CRUDGenUpdateBlob(dbo.TableName, dbo.Key.Value, dbo.BinaryData)
		} else {
			dbo.dbCtx.CRUDGenUpdate(dbo.TableName, dbo.Key.Value, SIMapToJson(dbo.Data))
		}
	}
}

func (dbo *DataBaseObject) Delete() {
	if dbo.Key != nil {
		logger.Info("delete dbo")
		dbo.dbCtx.CRUDGenDelete(dbo.TableName, dbo.Key.Value)
	}
}
