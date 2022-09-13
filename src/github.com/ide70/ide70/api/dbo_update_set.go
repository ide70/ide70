package api

import (
	"reflect"
	//	"strings"
)

type DboUpdateSet struct {
	dbos        map[string]*DataBaseObject
	updatedDBOs map[string]bool
	dbCtx       *DatabaseContext
}

func (dbCtx *DatabaseContext) CreateDBOUpdateSet(dboRoot *DataBaseObject) *DboUpdateSet {
	logger.Info("CreateDBOUpdateSet:", dboRoot)
	updateSet := &DboUpdateSet{dbCtx: dbCtx, dbos: map[string]*DataBaseObject{}, updatedDBOs: map[string]bool{}}
	updateSet.addDBO(dboRoot, "", 0)
	return updateSet
}

func (updateSet *DboUpdateSet) UpdateWithData(data map[string]interface{}) *DboUpdateSet {
	logger.Info("UpdateWithData:", data)
	IApplyFnToNodes(data, func(entry CollectionEntry) {
		switch siMap := entry.Value().(type) {
		case SIMap:
			logger.Info("auto converting key:", entry.LinearKey())
			entry.Update(map[string]interface{}(siMap))
		}
	})
	dboRoot := updateSet.dbos[""]
	updateSet.update(dboRoot.TableName, data, "", 0)
	for setId, dbo := range updateSet.dbos {
		if !updateSet.updatedDBOs[setId] {
			dbo.MarkToDelete()
		}
	}
	return updateSet
}

func (updateSet *DboUpdateSet) IsNew() bool {
	return updateSet.dbos[""].Key == nil
}

func (updateSet *DboUpdateSet) updateDBO(setId string, tableName string, data map[string]interface{}, saveOrder int) *DataBaseObject {
	logger.Info("updatedbo", setId, tableName)
	dbo := updateSet.dbos[setId]
	if dbo == nil {
		dbo = updateSet.dbCtx.CreateDBO(data, tableName)
		dbo.saveOrder = saveOrder
		updateSet.dbos[setId] = dbo
	} else {
		dbo.UpdateData(data)
	}
	for key, dbo := range updateSet.dbos {
		logger.Info("udbo", key, dbo)
	}
	updateSet.updatedDBOs[setId] = true

	return dbo
}

func (updateSet *DboUpdateSet) updateDBOBinary(setId string, tableName string, data *BinaryData, saveOrder int) *DataBaseObject {
	logger.Info("updatedbo binary", setId, tableName)
	dbo := updateSet.dbos[setId]
	if dbo == nil {
		dbo = updateSet.dbCtx.CreateBinaryDBO(data, tableName)
		dbo.saveOrder = saveOrder
		updateSet.dbos[setId] = dbo
	} else {
		dbo.UpdateBinaryData(data)
	}
	for key, dbo := range updateSet.dbos {
		logger.Info("udbo", key, dbo)
	}
	updateSet.updatedDBOs[setId] = true

	return dbo
}

func (updateSet *DboUpdateSet) addDBO(dbo *DataBaseObject, setId string, saveOrder int) {
	logger.Info("addDBO", setId)
	if dbo == nil {
		return
	}
	dbo.saveOrder = saveOrder
	updateSet.dbos[setId] = dbo
	for key, dbo := range updateSet.dbos {
		logger.Info("adbo", key, dbo)
	}
}

func (updateSet *DboUpdateSet) hasDBO(setId string) bool {
	for key, dbo := range updateSet.dbos {
		logger.Info("kdbo", key, dbo)
	}
	return updateSet.dbos[setId] != nil
}

func (updateSet *DboUpdateSet) update(tableName string, data map[string]interface{}, setIdPrefix string, baseSaveOrder int) *DataBaseObject {
	tableStruct := getTableStruct(tableName)
	dboRoot := updateSet.updateDBO(setIdPrefix, tableName, data, baseSaveOrder)
	dataKeysToDelete := map[string]bool{}
	logger.Info("update process loop")
	IApplyFnToNodes(data, func(entry CollectionEntry) {
		logger.Info("--LK:", entry.LinearKey())
		if entry.Parent() != nil {
			logger.Info("--PK:", entry.Parent().Key())
			if conn, has := tableStruct.connections[entry.Parent().LinearKey()]; has {
				switch entryT := entry.(type) {
				case *ArrayEntry:
					if conn.mxTableName != "" {
						logger.Info("entry.LinearKey():", entry.LinearKey())
						logger.Info("conn:", conn)
						logger.Info("entry:", reflect.TypeOf(entry.Value()))

						mxData := map[string]interface{}{}
						mxData["ord"] = entryT.Index()
						// set new DBO to refresh by other's generated IDs
						dboMx := updateSet.updateDBO(entry.LinearKey(), conn.mxTableName, mxData, baseSaveOrder+1)
						dboMx.addForeignKey(&ForeignKey{columnName: "id_" + tableName, foreignDBO: dboRoot})
						dboForeign := updateSet.update(conn.foreignTable.name, IAsSIMap(entry.Value()), setIdPrefix+entry.LinearKey()+".F", baseSaveOrder)
						dboMx.addForeignKey(&ForeignKey{columnName: "id_" + conn.foreignTable.name, foreignDBO: dboForeign})
					} else {
						logger.Info("E entry.LinearKey():", entry.LinearKey())
						logger.Info("E conn:", conn)
						logger.Info("E entry:", reflect.TypeOf(entry.Value()))

						dboForeign := updateSet.update(conn.foreignTable.name, IAsSIMap(entry.Value()), setIdPrefix+entry.LinearKey()+".F", baseSaveOrder+1)
						dboForeign.Data["_ord"] = entryT.Index()
						dboForeign.addForeignKey(&ForeignKey{columnName: conn.foreignColumn.name, foreignDBO: dboRoot})
					}
					dataKeysToDelete[entry.Parent().LinearKey()] = true
				case *MapEntry:
					if conn.mxTableName == "" {
						setIdForeign := setIdPrefix + entry.Parent().LinearKey() + ".F"
						logger.Info("S setId:", setIdForeign)
						if updateSet.updatedDBOs[setIdForeign] {
							return
						}
						logger.Info("S entry.LinearKey():", entry.LinearKey())
						logger.Info("S conn:", conn)
						logger.Info("S entry:", reflect.TypeOf(entry.Value()))

						dboForeign := updateSet.update(conn.foreignTable.name, IAsSIMap(entry.Parent().Value()), setIdForeign, baseSaveOrder+1)
						dboForeign.addForeignKey(&ForeignKey{columnName: conn.foreignColumn.name, foreignDBO: dboRoot})
					}
					dataKeysToDelete[entry.Parent().LinearKey()] = true
				}
			}

			if conn, has := tableStruct.connections[entry.Key()]; has {
				switch valueT := entry.Value().(type) {
				case *BinaryData:
					logger.Info("B conn:", conn)
					dboForeign := updateSet.updateDBOBinary(entry.LinearKey()+".F", conn.foreignTable.name, valueT, baseSaveOrder-1)
					// update entry.Value() where dboForeign saves
					dboRoot.addForeignKey(&ForeignKey{columnName: entry.LinearKey(), foreignDBO: dboForeign})
				}
			}
		}

	})
	for dataKeyToDelete := range dataKeysToDelete {
		var dataIf interface{} = data
		SIMapRemoveValue(dataKeyToDelete, &dataIf)
	}
	return dboRoot
}

func (updateSet *DboUpdateSet) DataLookup(key string) interface{} {
	dboRoot := updateSet.dbos[""]
	return updateSet.dataLookupFromDBO(dboRoot, TokenizeKeyExpr(key), 0)
}

func (updateSet *DboUpdateSet) dataLookupFromDBO(dboRoot *DataBaseObject, tokenizedKey []string, baseSaveOrder int) interface{} {
	logger.Info("DataLookupFromDBO, key:", tokenizedKey)
	if dboRoot == nil {
		return nil
	}
	tableStruct := getTableStruct(dboRoot.TableName)
	if conn, has := tableStruct.connections[KeyExprToken(tokenizedKey, 0)]; has {
		logger.Info("connectedKey")
		if len(tokenizedKey) == 1 {
			if conn.mxTableName != "" {
				qc := updateSet.dbCtx.QueryCtx()
				logger.Info("mxNrQuery")
				mxTable := qc.mxTable(conn.mxTableName, dboRoot.TableName, conn.foreignTable.name)
				logger.Info("mxNrTable")
				mxNr := qc.From(mxTable).Count().Where(mxTable["id_"+dboRoot.TableName].Equals(dboRoot.Key.Value)).OneRow().GetOneColumnForConvert().AsInt64()
				logger.Info("mxNr", mxNr)
				rv := make([]interface{}, mxNr)
				logger.Info("rv:", rv)
				return rv
			} else {
				qc := updateSet.dbCtx.QueryCtx()
				logger.Info("mxNrQuery")
				foreignTable := qc.Table(conn.foreignTable.name)
				logger.Info("foreignTable")
				foreignNr := qc.From(foreignTable).Count().Where(foreignTable[conn.foreignColumn.name].Equals(dboRoot.Key.Value)).OneRow().GetOneColumnForConvert().AsInt64()
				logger.Info("foreignNr", foreignNr)
				rv := make([]interface{}, foreignNr)
				logger.Info("rv:", rv)
				return rv
			}
		}
		if KeyExprTokenArrIdx(tokenizedKey, 1) >= 0 {
			setIdDboForeign := UnTokenizeKeyExpr(tokenizedKey[:2]) + ".F"
			dboForeign := updateSet.dbos[setIdDboForeign]

			if conn.mxTableName != "" {
				if dboForeign == nil {
					keyIndex := KeyExprTokenArrIdx(tokenizedKey, 1)
					dboMx := updateSet.dbCtx.FindDBObyCriteria(conn.mxTableName, &ColumnCriteria{column: "id_" + dboRoot.TableName, value: dboRoot.Key.Value},
						&ColumnCriteria{column: "ord", value: int64(keyIndex)})
					logger.Info("dboMx", dboMx)
					updateSet.addDBO(dboMx, UnTokenizeKeyExpr(tokenizedKey[:2]), baseSaveOrder+1)
					dboForeign = updateSet.dbCtx.FindDBO(conn.foreignTable.name, IAsInt64(dboMx.Data["id_"+conn.foreignTable.name]))
					logger.Info("dboForeign", dboForeign)
					updateSet.addDBO(dboForeign, setIdDboForeign, baseSaveOrder)
				}
				return updateSet.dataLookupFromDBO(dboForeign, tokenizedKey[2:], baseSaveOrder)
			} else {
				if dboForeign == nil {
					keyIndex := KeyExprTokenArrIdx(tokenizedKey, 1)
					dboForeign = updateSet.dbCtx.FindDBObyCriteria(conn.foreignTable.name, &ColumnCriteria{column: conn.foreignColumn.name, value: dboRoot.Key.Value},
						&ColumnCriteria{column: "_ord", value: int64(keyIndex)})
					logger.Info("dboForeign", dboForeign)
					updateSet.addDBO(dboForeign, setIdDboForeign, baseSaveOrder+1)
				}
				return updateSet.dataLookupFromDBO(dboForeign, tokenizedKey[2:], baseSaveOrder+1)
			}
		} else { // single connecting object
			setIdDboForeign := UnTokenizeKeyExpr(tokenizedKey[:1]) + ".F"
			dboForeign := updateSet.dbos[setIdDboForeign]
			if dboForeign == nil {
				dboForeign = updateSet.dbCtx.FindDBObyCriteria(conn.foreignTable.name, &ColumnCriteria{column: conn.foreignColumn.name, value: dboRoot.Key.Value})
				logger.Info("dboForeign", dboForeign)
				updateSet.addDBO(dboForeign, setIdDboForeign, baseSaveOrder+1)
			}
			return updateSet.dataLookupFromDBO(dboForeign, tokenizedKey[1:], baseSaveOrder+1)
		}
	}
	
	valueIf := SICollGetNode(UnTokenizeKeyExpr(tokenizedKey), dboRoot.Data)
	switch value := valueIf.(type) {
		case map[string]interface{}:
		logger.Info("return value is map, fields to auto convert lookup")
		for subKey, subValue := range value {
			if conn, has := tableStruct.connections[subKey]; has {
				// blob connection, replace with BinaryData (empty dbo with key, read data later on reqest)
				if conn.foreignTable.tableType == TABLETYPE_BLOB {
					logger.Info("blob conn")
					setIdDboForeign := UnTokenizeKeyExpr(tokenizedKey[:1]) + ".F"
					dboForeign := updateSet.dbos[setIdDboForeign]
					if dboForeign == nil {
						bd := &BinaryData{}
						bd.dbCtx = updateSet.dbCtx
						bd.tableName = conn.foreignTable.name
						bd.key = &DataBaseObjectKey{Value: IAsInt64(subValue)}
						logger.Info("binary data", bd)
						logger.Info("binary data key ", *bd.key)
						dboForeign = updateSet.dbCtx.CreateBinaryDBO(bd, conn.foreignTable.name)
						logger.Info("dboForeign blob", dboForeign)
						updateSet.addDBO(dboForeign, setIdDboForeign, baseSaveOrder-1)
					}
					value[subKey] = dboForeign.BinaryData
				}			
			}
		}
	}
	
				
				
	
	return valueIf
}

func (updateSet *DboUpdateSet) firstInSet() *DataBaseObject {
	for _, dbo := range updateSet.dbos {
		return dbo
	}
	return nil
}

func (updateSet *DboUpdateSet) Save() {
	if len(updateSet.dbos) == 0 {
		return
	}
	minSaveOrder := updateSet.firstInSet().saveOrder
	maxSaveOrder := minSaveOrder
	for _, dbo := range updateSet.dbos {
		if dbo.saveOrder < minSaveOrder {
			minSaveOrder = dbo.saveOrder
		}
		if dbo.saveOrder > maxSaveOrder {
			maxSaveOrder = dbo.saveOrder
		}
	}
	for i := minSaveOrder; i <= maxSaveOrder; i++ {
		for _, dbo := range updateSet.dbos {
			if dbo.IsMarkedToDelete() {
				continue
			}
			if dbo.saveOrder == i {
				dbo.UpdateForeignKeys()
				dbo.Save()
			}
		}
	}
	// remove unused DBOs
	for i := maxSaveOrder; i >= minSaveOrder; i-- {
		for _, dbo := range updateSet.dbos {
			if !dbo.IsMarkedToDelete() {
				continue
			}
			if dbo.saveOrder == i {
				dbo.Delete()
			}
		}
	}
}

/*
update set create (t table, m map)
	create dbo t, saveorder: 0
for each node of m
	m.fullkey.lastTag.is connection c in t
	  for each emt of node arr
		  create dbo: mx, saveorder: 1
		  update set create(connected by table of c, value of emt)
		  delete emt value
	  end
	end
end.
*/
