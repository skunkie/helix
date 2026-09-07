// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package flag

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

func RequiredString(raw string) (interface{}, error) {
	if raw == "" {
		return nil, errors.New("must not be empty")
	}
	return raw, nil
}

func StringEnum(values ...string) ParseFunc {
	return func(raw string) (interface{}, error) {
		if slices.Contains(values, raw) {
			return raw, nil
		}
		return raw, fmt.Errorf("must be one of %q", values)
	}
}

func IntList(raw string) (interface{}, error) {
	var ints []int

	if raw == "" {
		return ints, nil
	}

	for raw := range strings.SplitSeq(raw, ",") {
		i, err := strconv.Atoi(raw)
		if err != nil {
			return ints, err
		}
		ints = append(ints, i)
	}

	return ints, nil
}
