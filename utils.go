package main

import "strconv"

func ParseUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

func MustParseUint64(s string) uint64 {
	rv, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return rv
}
