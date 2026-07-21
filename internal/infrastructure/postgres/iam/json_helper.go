package iam

import "encoding/json"

// jsonUnmarshal is a tiny wrapper so permission_resolver.go doesn't need to import
// encoding/json directly (keeps the import list clean for the cached resolver).
func jsonUnmarshal(raw []byte, v any) error {
	return json.Unmarshal(raw, v)
}
