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
