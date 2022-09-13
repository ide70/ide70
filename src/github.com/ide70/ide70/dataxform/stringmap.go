package dataxform

import (

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