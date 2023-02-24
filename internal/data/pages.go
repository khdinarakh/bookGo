package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidPageFormat = errors.New("invalid pages format")

type Pages int32

func (r Pages) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d pages", r)
	quotedJSONValue := strconv.Quote(jsonValue) //to wrap it in double quotes

	return []byte(quotedJSONValue), nil
}

func (r *Pages) UnmarshalJSON(jsonValue []byte) error {

	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidPageFormat
	}

	parts := strings.Split(unquotedJSONValue, " ")

	if len(parts) != 2 || parts[1] != "pages" {
		return ErrInvalidPageFormat
	}

	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidPageFormat
	}

	*r = Pages(i)
	return nil
}
