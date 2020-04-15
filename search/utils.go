package search

import (
	"sort"
	"strconv"
)

func uint16to32s(in []uint16) (out []uint32) {
	for _, i := range in {
		out = append(out, uint32(i))
	}
	return
}

func toList(in map[string]bool) (out []string) {
	for k := range in {
		out = append(out, k)
	}
	sort.Strings(out)
	return
}

func fromHexUint16(input string) (uint16, error) {
	val, err := strconv.ParseUint(input, 16, 16)
	if err != nil {
		return 0, err
	}
	return uint16(val), nil
}

func fromHexUint32(input string) (uint32, error) {
	val, err := strconv.ParseUint(input, 16, 32)
	if err != nil {
		return 0, err
	}
	return uint32(val), nil
}
