package types

import (
	"encoding/json"
)

// isJSONObjectWithTopLevelKey returns true if the given bytes are a valid JSON object
// with exactly one top-level key that is contained in the list of allowed keys.
func isJSONObjectWithTopLevelKey(jsonBytes RawContractMessage, allowedKeys []string) (bool, error) {
	if err := jsonBytes.ValidateBasic(); err != nil {
		return false, err
	}

	var document interface{}
	if err := json.Unmarshal(jsonBytes, &document); err != nil {
		return false, err // invalid JSON
	}

	// Check if it's a map/object
	documentMap, ok := document.(map[string]interface{})
	if !ok {
		return false, nil // valid JSON but not an object
	}

	if len(documentMap) != 1 {
		return false, nil
	}

	// Loop is executed exactly once
	for topLevelKey := range documentMap {
		for _, allowedKey := range allowedKeys {
			if allowedKey == topLevelKey {
				return true, nil
			}
		}
		return false, nil
	}

	panic("Reached unreachable code. This is a bug.")
}
