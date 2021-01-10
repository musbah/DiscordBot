package main

import "strconv"

func stringToInt(ID string) uint64 {

	number, err := strconv.ParseUint(ID, 10, 64)
	if err != nil {
		//All IDs should be valid uint64 values, so this should not happen
		log.Errorf("Could not convert snowflake to uint64, %s", err)
	}

	return number
}

func intToString(ID uint64) string {
	return strconv.FormatUint(ID, 10)
}
