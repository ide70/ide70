package api

import (
	"github.com/ide70/ide70/dataxform"
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

func (updateSet *DboUpdateSet) addDBO(dbo *DataBaseObject, setId string, saveOrder int) {
	logger.Info("addDBO", setId)
	dbo.saveOrder = saveOrder
	updateSet.dbos[setId] = dbo
	for key, dbo := range updateSet.dbos {
		logger.Info("adbo", key, dbo)
	}
}

func (updateSet *DboUpdateSet) update(tableName string, data map[string]interface{}, setIdPrefix string, baseSaveOrder int) *DataBaseObject {
	tableStruct := getTableStruct(tableName)
	dboRoot := updateSet.updateDBO(setIdPrefix, tableName, data, baseSaveOrder)
	dataKeysToDelete := map[string]bool{}
	logger.Info("update process loop")
	dataxform.IApplyFnToNodes(data, func(entry dataxform.CollectionEntry) {
		if entry.Parent() != nil {
			if conn, has := tableStruct.connections[entry.Parent().LinearKey()]; has {
				switch entryT := entry.(type) {
				case *dataxform.ArrayEntry:
					if conn.mxTableName != "" {
						logger.Info("entry.LinearKey():", entry.LinearKey())
						logger.Info("conn:", conn)
						logger.Info("entry:", reflect.TypeOf(entry.Value()))

						mxData := map[string]interface{}{}
						mxData["ord"] = entryT.Index()
						// set new DBO to refresh by other's generated IDs
						dboMx := updateSet.updateDBO(entry.LinearKey(), conn.mxTableName, mxData, baseSaveOrder+1)
						dboMx.addForeignKey(&ForeignKey{columnName: "id_" + tableName, foreignDBO: dboRoot})
						dboForeign := updateSet.update(conn.foreignTable.name, dataxform.IAsSIMap(entry.Value()), setIdPrefix+entry.LinearKey()+".F", baseSaveOrder)
						dboMx.addForeignKey(&ForeignKey{columnName: "id_" + conn.foreignTable.name, foreignDBO: dboForeign})
					} else {
						logger.Info("E entry.LinearKey():", entry.LinearKey())
						logger.Info("E conn:", conn)
						logger.Info("E entry:", reflect.TypeOf(entry.Value()))

						dboForeign := updateSet.update(conn.foreignTable.name, dataxform.IAsSIMap(entry.Value()), setIdPrefix+entry.LinearKey()+".F", baseSaveOrder+1)
						dboForeign.Data["_ord"] = entryT.Index()
						dboForeign.addForeignKey(&ForeignKey{columnName: conn.foreignColumn.name, foreignDBO: dboRoot})
					}
					dataKeysToDelete[entry.Parent().LinearKey()] = true
				}
			}
		}
	})
	for dataKeyToDelete := range dataKeysToDelete {
		var dataIf interface{} = data
		dataxform.SIMapRemoveValue(dataKeyToDelete, &dataIf)
	}
	return dboRoot
}

func (updateSet *DboUpdateSet) DataLookup(key string) interface{} {
	dboRoot := updateSet.dbos[""]
	return updateSet.dataLookupFromDBO(dboRoot, dataxform.TokenizeKeyExpr(key), 0)
}

func (updateSet *DboUpdateSet) dataLookupFromDBO(dboRoot *DataBaseObject, tokenizedKey []string, baseSaveOrder int) interface{} {
	logger.Info("DataLookupFromDBO, key:", tokenizedKey)
	tableStruct := getTableStruct(dboRoot.TableName)
	if conn, has := tableStruct.connections[dataxform.KeyExprToken(tokenizedKey, 0)]; has {
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
		if dataxform.KeyExprTokenArrIdx(tokenizedKey, 1) >= 0 {
			setIdDboForeign := dataxform.UnTokenizeKeyExpr(tokenizedKey[:2]) + ".F"
			dboForeign := updateSet.dbos[setIdDboForeign]
			if dboForeign == nil {
				if conn.mxTableName != "" {
					keyIndex := dataxform.KeyExprTokenArrIdx(tokenizedKey, 1)
					dboMx := updateSet.dbCtx.FindDBObyCriteria(conn.mxTableName, &ColumnCriteria{column: "id_" + dboRoot.TableName, value: dboRoot.Key.Value},
						&ColumnCriteria{column: "ord", value: int64(keyIndex)})
					logger.Info("dboMx", dboMx)
					updateSet.addDBO(dboMx, dataxform.UnTokenizeKeyExpr(tokenizedKey[:2]), baseSaveOrder+1)
					dboForeign = updateSet.dbCtx.FindDBO(conn.foreignTable.name, dataxform.IAsInt64(dboMx.Data["id_"+conn.foreignTable.name]))
					logger.Info("dboForeign", dboForeign)
					updateSet.addDBO(dboForeign, setIdDboForeign, baseSaveOrder)
					return updateSet.dataLookupFromDBO(dboForeign, tokenizedKey[2:], baseSaveOrder)
				} else {
					keyIndex := dataxform.KeyExprTokenArrIdx(tokenizedKey, 1)
					dboForeign := updateSet.dbCtx.FindDBObyCriteria(dboRoot.TableName, &ColumnCriteria{column: conn.foreignColumn.name, value: dboRoot.Key.Value},
						&ColumnCriteria{column: "_ord", value: int64(keyIndex)})
					logger.Info("dboForeign", dboForeign)
					updateSet.addDBO(dboForeign, setIdDboForeign, baseSaveOrder+1)
					return updateSet.dataLookupFromDBO(dboForeign, tokenizedKey[2:], baseSaveOrder+1)
				}
			}
		}
	}
	return dataxform.SICollGetNode(dataxform.UnTokenizeKeyExpr(tokenizedKey), dboRoot.Data)
}

func (updateSet *DboUpdateSet) Save() {
	for i := 0; i <= 1; i++ {
		for _, dbo := range updateSet.dbos {
			if dbo.IsMarkedToDelete() {
				continue
			}
			if i > 0 {
				dbo.UpdateForeignKeys()
			}
			if dbo.saveOrder == i {
				dbo.Save()
			}
		}
	}
	// remove unused DBOs
	for i := 1; i >= 0; i-- {
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
