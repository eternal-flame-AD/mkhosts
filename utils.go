package main

import (
	"runtime"
	"strings"
)

func removeRepByLoop(slc []string) []string {
	result := []string{}
	for i := range slc {
		flag := true
		for j := range result {
			if slc[i] == result[j] {
				flag = false
				break
			}
		}
		if flag {
			result = append(result, slc[i])
		}
	}
	return result
}

func detectLineBreakFromString(s string) string {
	var LineBreak string
	switch {
	case strings.Contains(s, "\r\n"):
		LineBreak = "\r\n"
		break
	case strings.Contains(s, "\n"):
		LineBreak = "\n"
		break
	case strings.Contains(s, "\r"):
		LineBreak = "\r"
		break
	default:
		switch runtime.GOOS {
		case "windows":
			LineBreak = "\r\n"
		case "darwin":
			LineBreak = "\r"
		default:
			LineBreak = "\n"
		}
	}
	return LineBreak
}

func StringSliceOrEmpty(s interface{}) []string {
	if res, ok := s.([]string); ok {
		return res
	}
	return []string{}
}
