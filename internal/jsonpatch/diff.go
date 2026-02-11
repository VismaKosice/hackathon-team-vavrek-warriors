package jsonpatch

import (
	"strconv"
	"strings"
)

// Diff computes an RFC 6902 JSON Patch that transforms a into b.
// Both a and b should be the result of json.Unmarshal into interface{}.
// Path should be "" for the root document.
func Diff(a, b interface{}, path string) []map[string]interface{} {
	// Both nil — no change
	if a == nil && b == nil {
		return nil
	}
	// One is nil — replace
	if a == nil || b == nil {
		return []map[string]interface{}{replaceOp(path, b)}
	}

	// Try matching types
	aMap, aIsMap := a.(map[string]interface{})
	bMap, bIsMap := b.(map[string]interface{})
	if aIsMap && bIsMap {
		return diffObjects(aMap, bMap, path)
	}

	aArr, aIsArr := a.([]interface{})
	bArr, bIsArr := b.([]interface{})
	if aIsArr && bIsArr {
		return diffArrays(aArr, bArr, path)
	}

	// Different types or different primitive values
	if a != b {
		return []map[string]interface{}{replaceOp(path, b)}
	}

	return nil
}

func diffObjects(a, b map[string]interface{}, path string) []map[string]interface{} {
	var ops []map[string]interface{}

	// Removed keys (in a but not in b)
	for k, av := range a {
		if _, ok := b[k]; !ok {
			ops = append(ops, removeOp(path+"/"+escapeKey(k)))
			_ = av
		}
	}

	// Added and changed keys
	for k, bv := range b {
		childPath := path + "/" + escapeKey(k)
		av, inA := a[k]
		if !inA {
			ops = append(ops, addOp(childPath, bv))
		} else {
			sub := Diff(av, bv, childPath)
			ops = append(ops, sub...)
		}
	}

	return ops
}

func diffArrays(a, b []interface{}, path string) []map[string]interface{} {
	var ops []map[string]interface{}

	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	// Compare common elements
	for i := 0; i < minLen; i++ {
		sub := Diff(a[i], b[i], path+"/"+strconv.Itoa(i))
		ops = append(ops, sub...)
	}

	// Elements removed (reverse order to keep indices valid)
	for i := len(a) - 1; i >= minLen; i-- {
		ops = append(ops, removeOp(path+"/"+strconv.Itoa(i)))
	}

	// Elements added
	for i := minLen; i < len(b); i++ {
		ops = append(ops, addOp(path+"/"+strconv.Itoa(i), b[i]))
	}

	return ops
}

func replaceOp(path string, value interface{}) map[string]interface{} {
	return map[string]interface{}{"op": "replace", "path": path, "value": value}
}

func addOp(path string, value interface{}) map[string]interface{} {
	return map[string]interface{}{"op": "add", "path": path, "value": value}
}

func removeOp(path string) map[string]interface{} {
	return map[string]interface{}{"op": "remove", "path": path}
}

// escapeKey escapes a JSON Pointer token per RFC 6901.
func escapeKey(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}
