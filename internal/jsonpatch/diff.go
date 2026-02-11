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
	for k := range a {
		if _, ok := b[k]; !ok {
			ops = append(ops, removeOp(path+"/"+escapeKey(k)))
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

// DiffBoth computes both forward (a→b) and backward (b→a) patches in a single traversal.
func DiffBoth(a, b interface{}, path string) (fwd, bwd []map[string]interface{}) {
	if a == nil && b == nil {
		return nil, nil
	}
	if a == nil || b == nil {
		return []map[string]interface{}{replaceOp(path, b)},
			[]map[string]interface{}{replaceOp(path, a)}
	}

	aMap, aIsMap := a.(map[string]interface{})
	bMap, bIsMap := b.(map[string]interface{})
	if aIsMap && bIsMap {
		return diffObjectsBoth(aMap, bMap, path)
	}

	aArr, aIsArr := a.([]interface{})
	bArr, bIsArr := b.([]interface{})
	if aIsArr && bIsArr {
		return diffArraysBoth(aArr, bArr, path)
	}

	if a != b {
		return []map[string]interface{}{replaceOp(path, b)},
			[]map[string]interface{}{replaceOp(path, a)}
	}

	return nil, nil
}

func diffObjectsBoth(a, b map[string]interface{}, path string) (fwd, bwd []map[string]interface{}) {
	for k, av := range a {
		childPath := path + "/" + escapeKey(k)
		if _, ok := b[k]; !ok {
			fwd = append(fwd, removeOp(childPath))
			bwd = append(bwd, addOp(childPath, av))
		}
	}

	for k, bv := range b {
		childPath := path + "/" + escapeKey(k)
		av, inA := a[k]
		if !inA {
			fwd = append(fwd, addOp(childPath, bv))
			bwd = append(bwd, removeOp(childPath))
		} else {
			subFwd, subBwd := DiffBoth(av, bv, childPath)
			fwd = append(fwd, subFwd...)
			bwd = append(bwd, subBwd...)
		}
	}

	return fwd, bwd
}

func diffArraysBoth(a, b []interface{}, path string) (fwd, bwd []map[string]interface{}) {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	for i := 0; i < minLen; i++ {
		childPath := path + "/" + strconv.Itoa(i)
		subFwd, subBwd := DiffBoth(a[i], b[i], childPath)
		fwd = append(fwd, subFwd...)
		bwd = append(bwd, subBwd...)
	}

	// a has extra elements: forward removes (descending), backward adds (ascending)
	for i := len(a) - 1; i >= minLen; i-- {
		fwd = append(fwd, removeOp(path+"/"+strconv.Itoa(i)))
	}
	for i := minLen; i < len(a); i++ {
		bwd = append(bwd, addOp(path+"/"+strconv.Itoa(i), a[i]))
	}

	// b has extra elements: forward adds (ascending), backward removes (descending)
	for i := minLen; i < len(b); i++ {
		fwd = append(fwd, addOp(path+"/"+strconv.Itoa(i), b[i]))
	}
	for i := len(b) - 1; i >= minLen; i-- {
		bwd = append(bwd, removeOp(path+"/"+strconv.Itoa(i)))
	}

	return fwd, bwd
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
