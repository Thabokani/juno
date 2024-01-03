package utils

import "github.com/NethermindEth/juno/encoder"

// Clone deep copies an object by serializing and deserializing it
// Therefore it is limited to cloning public fields only.
func Clone[T any](v T) (T, error) {
	var copy T
	if encoded, err := encoder.Marshal(v); err != nil {
		return copy, err
	} else if err = encoder.Unmarshal(encoded, &copy); err != nil {
		return copy, err
	}
	return copy, nil
}
