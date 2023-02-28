package config

import "strconv"

// GCP requires string values inside the YAML file with environment values.
// For example, we need to use "0.25" instead of 0.25.
// The YAML parser can only decode "0.25" as a string. A custom type to support both is introduced.

type floatAsStr float64

func (f floatAsStr) MarshalText() ([]byte, error) {
	text := ""
	if f != 0 {
		text = strconv.FormatFloat(float64(f), 'f', -1, 64)
	}
	return []byte(text), nil
}

func (f *floatAsStr) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*f = 0
		return nil
	}
	value, err := strconv.ParseFloat(string(text), 64)
	if err != nil {
		return err
	}
	*f = floatAsStr(value)
	return nil
}
