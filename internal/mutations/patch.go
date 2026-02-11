package mutations

import (
	"time"

	json "github.com/goccy/go-json"
)

var emptyPatch = []byte("[]")
var jsonNull = json.RawMessage("null")

// fastParseDate parses "YYYY-MM-DD" ~10x faster than time.Parse by avoiding layout parsing.
// Returns zero time and false on invalid input.
func fastParseDate(s string) (time.Time, bool) {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return time.Time{}, false
	}
	y := int(s[0]-'0')*1000 + int(s[1]-'0')*100 + int(s[2]-'0')*10 + int(s[3]-'0')
	m := time.Month(int(s[5]-'0')*10 + int(s[6]-'0'))
	d := int(s[8]-'0')*10 + int(s[9]-'0')
	if m < 1 || m > 12 || d < 1 || d > 31 {
		return time.Time{}, false
	}
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), true
}

type patchOp struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value,omitempty"`
}

func marshalPatches(ops []patchOp) []byte {
	if len(ops) == 0 {
		return emptyPatch
	}
	b, _ := json.Marshal(ops)
	return b
}

func marshalValue(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
