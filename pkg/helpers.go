package pkg

import (
	"github.com/araddon/dateparse"
	"github.com/pkg/errors"
	"github.com/tj/go-naturaldate"
	"time"
)

func convertToStringList(value []interface{}) ([]string, error) {
	stringList := make([]string, len(value))
	for i, v := range value {
		s, ok := v.(string)
		if !ok {
			return nil, errors.Errorf("Not a string: %v", v)
		}
		stringList[i] = s
	}
	return stringList, nil
}

func parseDate(value string) (time.Time, error) {
	parsedDate, err := dateparse.ParseAny(value)
	if err != nil {
		refTime_ := time.Now()
		if refTime != nil {
			refTime_ = *refTime
		}
		parsedDate, err = naturaldate.Parse(value, refTime_)
		if err != nil {
			return time.Time{}, errors.Wrapf(err, "Could not parse date: %s", value)
		}
	}

	return parsedDate, nil
}
