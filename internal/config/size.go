package config

import (
	"fmt"
	"strconv"
	"strings"
)

type ByteSize int64

func (b ByteSize) Int64() int64 {
	return int64(b)
}

func (b *ByteSize) UnmarshalYAML(unmarshal func(any) error) error {
	var raw any
	if err := unmarshal(&raw); err != nil {
		return err
	}

	switch v := raw.(type) {
	case int:
		*b = ByteSize(v)
		return nil
	case int64:
		*b = ByteSize(v)
		return nil
	case uint64:
		*b = ByteSize(v)
		return nil
	case string:
		size, err := ParseByteSize(v)
		if err != nil {
			return err
		}
		*b = size
		return nil
	default:
		return fmt.Errorf("invalid size value %T", raw)
	}
}

func ParseByteSize(value string) (ByteSize, error) {
	input := strings.TrimSpace(value)
	if input == "" {
		return 0, fmt.Errorf("size must not be empty")
	}

	input = strings.ReplaceAll(input, " ", "")
	split := 0
	for split < len(input) && ((input[split] >= '0' && input[split] <= '9') || input[split] == '.') {
		split++
	}

	if split == 0 {
		return 0, fmt.Errorf("size %q must start with a number", value)
	}

	numberPart := input[:split]
	unitPart := strings.ToUpper(input[split:])
	if unitPart == "" {
		unitPart = "B"
	}

	number, err := strconv.ParseFloat(numberPart, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric size %q: %w", value, err)
	}
	if number <= 0 {
		return 0, fmt.Errorf("size %q must be greater than zero", value)
	}

	multiplier, ok := sizeUnits[unitPart]
	if !ok {
		return 0, fmt.Errorf("unsupported size unit %q", unitPart)
	}

	size := int64(number * float64(multiplier))
	if size <= 0 {
		return 0, fmt.Errorf("size %q must be greater than zero", value)
	}

	return ByteSize(size), nil
}

var sizeUnits = map[string]int64{
	"B":   1,
	"KB":  1000,
	"MB":  1000 * 1000,
	"GB":  1000 * 1000 * 1000,
	"TB":  1000 * 1000 * 1000 * 1000,
	"KIB": 1024,
	"MIB": 1024 * 1024,
	"GIB": 1024 * 1024 * 1024,
	"TIB": 1024 * 1024 * 1024 * 1024,
}
