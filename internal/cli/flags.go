package cli

import (
	"database/sql"
	"flag"
	"fmt"
	"time"
)

// The update queries leave a column alone when its parameter is NULL, so an
// unset flag maps to an invalid sql.Null value. A zero value means "unset",
// which is what the flags already implied.

func optStr(v string) sql.NullString {
	return sql.NullString{String: v, Valid: v != ""}
}

func optInt(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: v != 0}
}

func optFloat(v float64) sql.NullFloat64 {
	return sql.NullFloat64{Float64: v, Valid: v != 0}
}

// wasSet reports which flags actually appeared on the command line. Leaving an
// update flag out means "leave the column alone", which is not the same as
// passing it empty to clear the column. Only meaningful after Parse.
func wasSet(fs *flag.FlagSet) map[string]bool {
	set := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	return set
}

func setStr(v string, ok bool) sql.NullString { return sql.NullString{String: v, Valid: ok} }

func setBool(v, ok bool) sql.NullBool { return sql.NullBool{Bool: v, Valid: ok} }

const timeLayout = "2006-01-02 15:04:05"

func parseTime(label, v string) (time.Time, error) {
	t, err := time.ParseInLocation(timeLayout, v, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: expected %q, got %q", label, timeLayout, v)
	}
	return t, nil
}

func optTime(label, v string) (sql.NullInt64, error) {
	if v == "" {
		return sql.NullInt64{}, nil
	}
	t, err := parseTime(label, v)
	if err != nil {
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: t.Unix(), Valid: true}, nil
}

func required(name, v string) error {
	if v == "" {
		return fmt.Errorf("-%s is required", name)
	}
	return nil
}
