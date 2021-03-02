package dataxform

import ()

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

func SIMapGetByKeyAsList(m map[string]interface{}, k string) []interface{} {
	entry := m[k]
	if entry == nil {
		return []interface{}{}
	}
	return entry.([]interface{})
}

func SIMapGetByKeyAsMap(m map[string]interface{}, k string) map[string]interface{} {
	entry := m[k]
	if entry == nil {
		return map[string]interface{}{}
	}
	return entry.(map[string]interface{})
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

/*
// test
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
