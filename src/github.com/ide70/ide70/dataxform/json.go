package dataxform

import (
	"bytes"
	"encoding/json"
	"strings"
)

func SIMapToJson(m map[string]interface{}) string {
	dstBuf := bytes.NewBufferString("")
	encoder := json.NewEncoder(dstBuf)
	encoder.Encode(m)
	return dstBuf.String()
}

func SIMapToJsonPP(m map[string]interface{}) string {
	dstBuf := bytes.NewBufferString("")
	encoder := json.NewEncoder(dstBuf)
	encoder.SetIndent("", "  ")
	encoder.Encode(m)
	return dstBuf.String()
}

func JsonToSIMap(data string) map[string]interface{} {
	dataMap := map[string]interface{}{}
	decoder := json.NewDecoder(strings.NewReader(data))
	decoder.Decode(&dataMap)
	return dataMap
}
