package api

import (
	"fmt"
	"reflect"
	"strings"
)

func InterfaceReplaceMapKeyToString(i interface{}) interface{} {
	switch itp := i.(type) {
	case []interface{}:
		return InterfaceListReplaceMapKeyToString(itp)
	case map[interface{}]interface{}:
		return InterfaceMapToStringMap(itp)
	}
	return i
}

func InterfaceListReplaceMapKeyToString(l []interface{}) []interface{} {
	dl := []interface{}{}
	for _, e := range l {
		dl = append(dl, InterfaceReplaceMapKeyToString(e))
	}
	return dl
}

func InterfaceMapToStringMap(m map[interface{}]interface{}) map[string]interface{} {
	dm := map[string]interface{}{}
	for k, v := range m {
		dm[k.(string)] = InterfaceReplaceMapKeyToString(v)
	}
	return dm
}

func IArrToStringArr(list []interface{}) []string {
	listS := []string{}
	for _, item := range list {
		listS = append(listS, IAsString(item))
	}
	return listS
}

func SIMapGetByKeyAsList(m map[string]interface{}, k string) []interface{} {
	entry := m[k]
	if entry == nil {
		return []interface{}{}
	}
	return entry.([]interface{})
}

func SIMapGetByKeyAsMap(m map[string]interface{}, k string) map[string]interface{} {
	if m == nil {
		return map[string]interface{}{}
	}
	entry := m[k]
	if entry == nil {
		return map[string]interface{}{}
	}
	return entry.(map[string]interface{})
}

func SIMapGetByKeyIsString(m map[string]interface{}, k string) bool {
	if m == nil {
		return false
	}
	entry := m[k]
	switch entry.(type) {
	case string:
		return true
	}
	return false
}

func SIMapGetByKeyChainAsMap(m map[string]interface{}, k string) map[string]interface{} {
	keys := strings.Split(k, ".")
	for _, key := range keys {
		m = SIMapGetByKeyAsMap(m, key)
	}
	return m
}

func SIMapGetByKeyChain(m map[string]interface{}, k string) interface{} {
	keys := strings.Split(k, ".")
	lastIdx := len(keys) - 1
	for idx, key := range keys {
		if idx == lastIdx {
			return m[key]
		}
		m = SIMapGetByKeyAsMap(m, key)
	}
	return m
}

func AsSIMap(i interface{}) map[string]interface{} {
	return i.(map[string]interface{})
}

func SIMapGetByKeyAsString(m map[string]interface{}, k string) string {
	entry := m[k]
	if entry == nil {
		return ""
	}
	return entry.(string)
}

func SIMapGetByKeyAsBoolean(m map[string]interface{}, k string) bool {
	entry := m[k]
	if entry == nil {
		return false
	}
	return entry.(bool)
}

func SIMapCopyKeys(s, d map[string]interface{}, keys []string) {
	for _, key := range keys {
		if _, has := s[key]; has {
			d[key] = s[key]
		}
	}
}

func IAsString(i interface{}) string {
	if i == nil {
		return ""
	}
	switch iT := i.(type) {
	case int, int64:
		return fmt.Sprintf("%d", iT)
	case string:
		return iT
	}
	return ""
}

func IIsInt(i interface{}) bool {
	if i == nil {
		return false
	}
	switch i.(type) {
	case int:
		return true
	case int64:
		return true
	}
	return false
}

func IIsNumber(i interface{}) bool {
	if i == nil {
		return false
	}
	switch i.(type) {
	case int:
		return true
	case int8:
		return true
	case int16:
		return true
	case int32:
		return true
	case int64:
		return true
	case float32:
		return true
	case float64:
		return true
	}
	return false
}

func IAsInt(i interface{}) int {
	if i == nil {
		return 0
	}
	switch iT := i.(type) {
	case int:
		return iT
	case int64:
		return int(iT)
	case float64:
		return int(iT)
	case string:
		iC := 0
		fmt.Sscanf(iT, "%d", &iC)
		return iC
	}
	logger.Warning("IAsInt unknown type:", reflect.TypeOf(i))
	return 0
}

func IAsInt64(i interface{}) int64 {
	if i == nil {
		return 0
	}
	switch iT := i.(type) {
	case int:
		return int64(iT)
	case int64:
		return iT
	case float64:
		return int64(iT)
	case string:
		var iC int64 = 0
		fmt.Sscanf(iT, "%d", &iC)
		return iC
	}
	logger.Warning("IAsInt unknown type:", reflect.TypeOf(i))
	return 0
}

func IAsBool(i interface{}) bool {
	if i == nil {
		return false
	}
	switch iT := i.(type) {
	case int, int64:
		return iT == 1
	case bool:
		return iT
	case string:
		return iT == "true" || iT == "yes"
	}
	return false
}

func IAsArr(i interface{}) []interface{} {
	if i == nil {
		return []interface{}{}
	}
	switch iT := i.(type) {
	case []interface{}:
		return iT
	}
	return []interface{}{}
}

func IAsSIMap(i interface{}) map[string]interface{} {
	if i == nil {
		return map[string]interface{}{}
	}

	switch iT := i.(type) {
	case map[string]interface{}:
		return iT
	}
	return map[string]interface{}{}
}

func SIMapGetByKeyAsInt(m map[string]interface{}, k string) int {
	entry := m[k]
	if entry == nil {
		return 0
	}
	return entry.(int)
}

// override mbase map with values in mover recursively
func SIMapOverride(mBase, mOver map[string]interface{}) {
	for k, v := range mOver {
		if _, has := mBase[k]; has {
			switch vBaseT := mBase[k].(type) {
			case map[string]interface{}:
				switch vT := v.(type) {
				case map[string]interface{}:
					SIMapOverride(vBaseT, vT)
				}
			case []interface{}:
				switch vT := v.(type) {
				case []interface{}:
					mBase[k] = append(vBaseT, vT...)
				}
			default:
				mBase[k] = v
			}
		} else {
			mBase[k] = v
		}
	}
}

// inject or join mDefs maps values into mOver map recursively
func SIMapInjectDefaults(mDefs, mOver map[string]interface{}) {
	for k, v := range mDefs {
		if _, has := mOver[k]; has {
			switch vOverT := mOver[k].(type) {
			case map[string]interface{}:
				switch vT := v.(type) {
				case map[string]interface{}:
					SIMapInjectDefaults(vT, vOverT)
				}
			case []interface{}:
				switch vT := v.(type) {
				case []interface{}:
					mOver[k] = append(vT, vOverT...)
				}
			}
		} else {
			mOver[k] = v
		}
	}
}

type MapEntry struct {
	parent CollectionEntry
	m      map[string]interface{}
	k      string
	v      interface{}
	stop   bool
}

type ArrayEntry struct {
	parent CollectionEntry
	a      []interface{}
	i      int
	v      interface{}
	stop   bool
}

type CollectionEntry interface {
	Key() string
	Index() int
	Value() interface{}
	SameLevelValue(string) interface{}
	Delete()
	Update(v interface{})
	LinearKey() string
	Parent() CollectionEntry
	Stop()
}

func (entry *MapEntry) Delete() {
	delete(entry.m, entry.k)
}

func (entry *MapEntry) Stop() {
	entry.stop = true
	if entry.parent != nil {
		entry.parent.Stop()
	}
}

func (entry *MapEntry) Update(v interface{}) {
	entry.m[entry.k] = v
}

func (entry *MapEntry) Key() string {
	return entry.k
}

func (entry *MapEntry) Index() int {
	return -1
}

func (entry *MapEntry) Value() interface{} {
	return entry.v
}

func (entry *MapEntry) SameLevelValue(key string) interface{} {
	return entry.m[key]
}

func (entry *ArrayEntry) SameLevelValue(key string) interface{} {
	return nil
}

func (entry *MapEntry) Parent() CollectionEntry {
	return entry.parent
}

func (entry *MapEntry) LinearKey() string {
	linearKey := ""
	if entry.parent != nil {
		linearKey = entry.parent.LinearKey()
		linearKey += "."
	}
	return linearKey + entry.k
}

func (entry *ArrayEntry) LinearKey() string {
	linearKey := ""
	if entry.parent != nil {
		linearKey = entry.parent.LinearKey()
	}
	return linearKey + fmt.Sprintf("[%d]", entry.i)
}

func (entry *ArrayEntry) Delete() {
	entry.a[entry.i] = nil
}

func (entry *ArrayEntry) Update(v interface{}) {
	entry.a[entry.i] = v
}

func (entry *ArrayEntry) Key() string {
	return ""
}

func (entry *ArrayEntry) Index() int {
	return entry.i
}

func (entry *ArrayEntry) Value() interface{} {
	return entry.v
}

func (entry *ArrayEntry) Parent() CollectionEntry {
	return entry.parent
}

func (entry *ArrayEntry) Stop() {
	entry.stop = true
	if entry.parent != nil {
		entry.parent.Stop()
	}
}

func IApplyFn(i interface{}, f func(entry CollectionEntry)) {
	switch iT := i.(type) {
	case map[string]interface{}:
		SIMapApplyFn(iT, f)
	case []interface{}:
		IArrApplyFn(iT, f)
	}
}

func IApplyFnToNodes(i interface{}, f func(entry CollectionEntry)) {
	switch iT := i.(type) {
	case map[string]interface{}:
		SIMapApplyFnToNodes(iT, f)
	case []interface{}:
		IArrApplyFnToNodes(iT, f)
	}
}

func IArrApplyFnLinear(a []interface{}, f func(entry CollectionEntry)) {
	iArrApplyFnLinear(a, f, nil)
}

func SIMapApplyFn(m map[string]interface{}, f func(entry CollectionEntry)) {
	sIMapApplyFn(m, f, nil)
}

func IArrApplyFnToNodes(a []interface{}, f func(entry CollectionEntry)) {
	iArrApplyFnToNodes(a, f, nil)
}

func IArrApplyFn(a []interface{}, f func(entry CollectionEntry)) {
	iArrApplyFnToNodes(a, f, nil)
}

func SIMapApplyFnToNodes(m map[string]interface{}, f func(entry CollectionEntry)) {
	sIMapApplyFnToNodes(m, f, nil)
}

func iArrApplyFnLinear(a []interface{}, f func(entry CollectionEntry), parentEntry CollectionEntry) {
	for i, v := range a {
		entry := &ArrayEntry{parent: parentEntry, a: a, i: i, v: v}
		f(entry)
		if entry.stop {
			return
		}
	}
}

func iArrApplyFn(a []interface{}, f func(entry CollectionEntry), parentEntry CollectionEntry) {
	for i, v := range a {
		entry := &ArrayEntry{parent: parentEntry, a: a, i: i, v: v}
		switch vT := v.(type) {
		case SIMap:
			sIMapApplyFn(vT, f, entry)
			if entry.stop {
				return
			}
		case map[string]interface{}:
			sIMapApplyFn(vT, f, entry)
			if entry.stop {
				return
			}
		case []interface{}:
			iArrApplyFn(vT, f, entry)
			if entry.stop {
				return
			}
		default:
			f(entry)
			if entry.stop {
				return
			}
		}
	}
}

func iArrApplyFnToNodes(a []interface{}, f func(entry CollectionEntry), parentEntry CollectionEntry) {
	for i, v := range a {
		entry := &ArrayEntry{parent: parentEntry, a: a, i: i, v: v}
		switch vT := v.(type) {
		case SIMap:
			f(entry)
			if entry.stop {
				return
			}
			sIMapApplyFnToNodes(vT, f, entry)
		case map[string]interface{}:
			f(entry)
			if entry.stop {
				return
			}
			sIMapApplyFnToNodes(vT, f, entry)
		case []interface{}:
			f(entry)
			if entry.stop {
				return
			}
			iArrApplyFnToNodes(vT, f, entry)
		default:
			f(entry)
			if entry.stop {
				return
			}
		}
	}
}

// apply func on map leafs
func sIMapApplyFn(m map[string]interface{}, f func(entry CollectionEntry), parentEntry CollectionEntry) {
	for k, v := range m {
		entry := &MapEntry{parent: parentEntry, m: m, k: k, v: v}
		switch vT := v.(type) {
		case SIMap:
			sIMapApplyFn(vT, f, entry)
			if entry.stop {
				return
			}
		case map[string]interface{}:
			sIMapApplyFn(vT, f, entry)
			if entry.stop {
				return
			}
		case []interface{}:
			iArrApplyFn(vT, f, entry)
			if entry.stop {
				return
			}
		default:
			f(entry)
			if entry.stop {
				return
			}
		}
	}
}

func sIMapApplyFnToNodes(m map[string]interface{}, f func(entry CollectionEntry), parentEntry CollectionEntry) {
	for k, v := range m {
		entry := &MapEntry{parent: parentEntry, m: m, k: k, v: v}
		switch vT := v.(type) {
		case SIMap:
			f(entry)
			if entry.stop {
				return
			}
			sIMapApplyFnToNodes(vT, f, entry)
		case map[string]interface{}:
			f(entry)
			if entry.stop {
				return
			}
			sIMapApplyFnToNodes(vT, f, entry)
		case []interface{}:
			f(entry)
			if entry.stop {
				return
			}
			iArrApplyFnToNodes(vT, f, entry)
		default:
			f(entry)
			if entry.stop {
				return
			}
		}
	}
}

// tests
/*func Tst_SIMapApplyFn() {
	m := map[string]interface{}{}
	m["akey"] = "avalue"
	m["bkey"] = "bvalue"
	m["ckey"] = []interface{}{"c1", "c2"}
	m["dkey"] = []interface{}{"d1", "d2"}
	m["ekey"] = map[string]interface{}{"e1k": "e1v", "e2k": "e2v"}
	m["fkey"] = map[string]interface{}{"f1k": "f1v", "f2k": "f2v"}

	fmt.Println(m)
	SIMapApplyFn(m, func(entry CollectionEntry) {
		if entry.Value().(string) == "bvalue" {
			entry.Update("bvalueMOD")
		}
		if entry.Value().(string) == "c2" {
			entry.Update("c2MOD")
		}
		if entry.Value().(string) == "e1v" {
			entry.Update("e1vMOD")
		}
		if entry.Value().(string) == "e2v" {
			entry.Delete()
		}
		if entry.Value().(string) == "d1" {
			entry.Delete()
		}
	})
	fmt.Println(m)
}

func tst1() {
	defs := map[string]interface{}{}
	over := map[string]interface{}{}
	defs["akey"] = "avalue"
	defs["bkey"] = "bvalue"
	defs["ckey"] = []interface{}{"c1", "c2"}
	defs["dkey"] = []interface{}{"d1", "d2"}
	defs["ekey"] = map[string]interface{}{"e1k": "e1v", "e2k": "e2v"}
	defs["fkey"] = map[string]interface{}{"f1k": "f1v", "f2k": "f2v"}
	over["aXkey"] = "aOvalue"
	over["bkey"] = "bOvalue"
	over["ckey"] = []interface{}{"cO3", "cO4"}
	over["dXkey"] = []interface{}{"dO1", "dO2"}
	over["ekey"] = map[string]interface{}{"e1k": "e1Ov", "e3k": "eO2v"}
	over["fXkey"] = map[string]interface{}{"f1k": "fO1v", "f2k": "fO2v"}

	fmt.Println(over)
	dataxform.SIMapInjectDefaults(defs, over)
	fmt.Println(over)
}
*/

func TokenizeKeyExpr(keyExpr string) []string {
	if strings.Contains(keyExpr, "[") {
		keyExpr = strings.Replace(keyExpr, "[", ".[", -1)
		keyExpr = strings.Replace(keyExpr, "]", "", -1)
	}
	if strings.HasPrefix(keyExpr, ".") {
		keyExpr = strings.TrimPrefix(keyExpr, ".")
	}
	return strings.Split(keyExpr, ".")
}

func UnTokenizeKeyExpr(tokenizedKey []string) string {
	var key strings.Builder
	for idx, token := range tokenizedKey {
		if strings.HasPrefix(token, "[") {
			key.WriteString(token)
			key.WriteString("]")
		} else {
			if idx > 0 {
				key.WriteString(".")
			}
			key.WriteString(token)
		}
	}
	return key.String()
}

func KeyExprToken(tokenizedKey []string, idx int) string {
	if idx >= len(tokenizedKey) {
		return ""
	}
	return tokenizedKey[idx]
}

func KeyExprTokenArrIdx(tokenizedKey []string, idx int) int {
	token := KeyExprToken(tokenizedKey, idx)
	return getIndexOfToken(token)
}

func SICollGetNode(keyExpr string, collection interface{}) interface{} {
	keyTokens := TokenizeKeyExpr(keyExpr)
	return sIMapGetNodeLevel(keyTokens, collection)
}

func sIMapGetNodeLevel(keyTokens []string, collection interface{}) interface{} {
	lastToken := len(keyTokens) == 1
	keyToken := keyTokens[0]
	index := getIndexOfToken(keyToken)

	if index >= 0 {
		slice := collection.([]interface{})
		if lastToken && index < len(slice) {
			return slice[index]
		}
		if !lastToken && index < len(slice) && slice[index] != nil {
			return sIMapGetNodeLevel(keyTokens[1:], slice[index])
		}
	} else {
		collMap := collection.(map[string]interface{})
		subElement, hasSubElement := collMap[keyToken]

		if lastToken {
			return subElement
		}
		if hasSubElement {
			return sIMapGetNodeLevel(keyTokens[1:], subElement)
		}
	}
	return nil
}

func SIMapCopy(collection interface{}) interface{} {
	switch c := collection.(type) {
	case map[string]interface{}:
		m := map[string]interface{}{}
		for k, v := range c {
			m[k] = SIMapCopy(v)
		}
		return m
	case []interface{}:
		a := []interface{}{}
		for _, v := range c {
			a = append(a, SIMapCopy(v))
		}
		return a
	default:
		return c
	}
}

func SIMapLightCopy(sm map[string]interface{}) map[string]interface{} {
	m := map[string]interface{}{}
	for k, v := range sm {
		m[k] = v
	}
	return m
}

func SIMapUpdateValue(keyExpr string, value interface{}, m map[string]interface{}, removeEmpty bool) {
	keyTokens := TokenizeKeyExpr(keyExpr)
	collection := interface{}(m)

	if removeEmpty && IsEmpty(value) {
		sIMapRemoveValueLevel(keyTokens, &collection)
	} else {
		sIMapAddValueLevel(keyTokens, value, &collection)
	}
}

func sIMapAddValueLevel(keyTokens []string, value interface{}, pCollection *interface{}) {
	collection := *pCollection
	lastToken := len(keyTokens) == 1
	keyToken := keyTokens[0]
	index := getIndexOfToken(keyToken)

	if index >= 0 {
		slice := collection.([]interface{})
		if !lastToken && index < len(slice) && slice[index] != nil {
			sIMapAddValueLevel(keyTokens[1:], value, &slice[index])
			*pCollection = slice
		} else {
			*pCollection = setSliceAt(slice, index, buildSiMapFromKeyTokensAndValue(keyTokens[1:], value))
		}

	} else {
		collMap := collection.(map[string]interface{})
		subElement, hasSubElement := collMap[keyToken]

		if !lastToken && hasSubElement {
			sIMapAddValueLevel(keyTokens[1:], value, &subElement)
			collMap[keyToken] = subElement
			*pCollection = collMap
		} else {
			collMap[keyToken] = buildSiMapFromKeyTokensAndValue(keyTokens[1:], value)
			*pCollection = collMap
		}
	}
}

func SIArrRemoveValue(keyExpr string, c *[]interface{}) {
	keyTokens := TokenizeKeyExpr(keyExpr)
	collection := interface{}(*c)
	sIMapRemoveValueLevel(keyTokens, &collection)
	*c = collection.([]interface{})
}

func SIMapRemoveValue(keyExpr string, pM *interface{}) {
	keyTokens := TokenizeKeyExpr(keyExpr)
	sIMapRemoveValueLevel(keyTokens, pM)
}

func sIMapRemoveValueLevel(keyTokens []string, pCollection *interface{}) bool {
	collection := *pCollection
	lastToken := len(keyTokens) == 1
	keyToken := keyTokens[0]
	index := getIndexOfToken(keyToken)

	if index >= 0 {
		slice := collection.([]interface{})
		if index < len(slice) {
			subElement := slice[index]
			if subElement != nil {
				if lastToken || sIMapRemoveValueLevel(keyTokens[1:], &subElement) {
					SliceDeleteAt(&slice, index)
					collection = slice
					*pCollection = collection
					return len(slice) == 0
				}
			}
		}
	} else {
		collMap := collection.(map[string]interface{})
		subElement, hasSubElement := collMap[keyToken]

		if hasSubElement {
			if lastToken || sIMapRemoveValueLevel(keyTokens[1:], &subElement) {
				delete(collMap, keyToken)
				*pCollection = collMap
				return len(collMap) == 0
			}
			collMap[keyToken] = subElement
			*pCollection = collMap
		}
	}

	return false
}

func getIndexOfToken(keyToken string) int {
	index := -1
	fmt.Sscanf(keyToken, "[%d", &index)
	return index
}

func SliceDeleteAt(sp *[]interface{}, i int) {
	*sp = append((*sp)[:i], (*sp)[i+1:]...)
}

func buildSiMapFromKeyTokensAndValue(keyTokens []string, value interface{}) interface{} {
	collection := value
	for idx := len(keyTokens) - 1; idx >= 0; idx-- {
		keyToken := keyTokens[idx]
		if strings.HasPrefix(keyToken, "[") {
			index := 0
			fmt.Sscanf(keyToken, "[%d", &index)
			collection = []interface{}{}
			collection = setSliceAt(collection.([]interface{}), index, value)
		} else {
			collection = map[string]interface{}{}
			collection.(map[string]interface{})[keyToken] = value
		}
		value = collection
	}
	return collection
}

func IsEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return v == ""
	}
	return false
}

func setSliceAt(slice []interface{}, index int, value interface{}) []interface{} {
	for len(slice) < index {
		slice = append(slice, nil)
	}
	if len(slice) > index {
		slice[index] = value
	} else {
		slice = append(slice, value)
	}
	return slice
}

func GetOnlyEntry(m map[string]interface{}) (string, interface{}) {
	for k, v := range m {
		return k, v
	}
	return "", nil
}

func StringListToSet(list []string) map[string]bool {
	m := map[string]bool{}
	for _, e := range list {
		m[e] = true
	}
	return m
}
