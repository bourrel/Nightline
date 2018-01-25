package svcdb

import "fmt"

func arrayToString(array []string) string {
	var str = "["

	for i := 0; i < len(array); i++ {
		if i != 0 {
			str += ", "
		}
		str += "\"" + array[i] + "\""
	}
	str += "]"

	fmt.Println(str)
	return str
}

func StrArrayToIArray(slice []string) []interface{} {
	ret := make([]interface{}, len(slice))

	for i := range slice {
		ret[i] = slice[i]
	}
	return ret
}
