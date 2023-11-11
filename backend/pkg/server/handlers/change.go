package handlers

import (
	"strconv"
)

func GetInterfaceToInt(t1 interface{}) int {
	var t2 int
	switch t1.(type) {
	case int8:
		t2 = int(t1.(int8))
		break
	case uint8:
		t2 = int(t1.(uint8))
		break
	case int16:
		t2 = int(t1.(int16))
		break
	case uint16:
		t2 = int(t1.(uint16))
		break
	case int32:
		t2 = int(t1.(int32))
		break
	case uint32:
		t2 = int(t1.(uint32))
		break
	case int64:
		t2 = int(t1.(int64))
		break
	case uint64:
		t2 = int(t1.(uint64))
		break
	case float32:
		t2 = int(t1.(float32))
		break
	case float64:
		t2 = int(t1.(float64))
		break
	case nil:
		t2 = 0
		break
	case string:
		t2, _ = strconv.Atoi(t1.(string))
		if t2 == 0 && len(t1.(string)) > 0 {
			f, _ := strconv.ParseFloat(t1.(string), 64)
			t2 = int(f)
		}
		break
	default:
		t2 = t1.(int)
		break
	}
	return t2
}
