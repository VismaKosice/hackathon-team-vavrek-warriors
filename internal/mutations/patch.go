package mutations

import json "github.com/goccy/go-json"

var emptyPatch = []byte("[]")
var jsonNull = json.RawMessage("null")

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
