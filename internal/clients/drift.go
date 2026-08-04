package clients

import (
	"reflect"
	"time"
)

// StringSlice converts a Vault response value of type []interface{} to []string,
// discarding any non-string elements.
func StringSlice(in []interface{}) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// StringMap converts a Vault response value of type map[string]interface{} to
// map[string]string, discarding any non-string values.
func StringMap(in map[string]interface{}) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

// DriftedString reports whether the observed Vault response value for key
// differs from want. It returns false when want is empty, or when the observed
// value is missing or of an unexpected type, so that Vault-injected defaults
// never trigger a spurious update loop.
func DriftedString(data map[string]interface{}, key, want string) bool {
	if want == "" {
		return false
	}
	got, ok := data[key].(string)
	return ok && got != want
}

// DriftedBool reports whether the observed Vault response value for key differs
// from want. It returns false when want is nil or the observed value is absent
// or not a bool.
func DriftedBool(data map[string]interface{}, key string, want *bool) bool {
	if want == nil {
		return false
	}
	got, ok := data[key].(bool)
	return ok && got != *want
}

// DriftedInt reports whether the observed Vault response value for key differs
// from want. Vault returns integers as JSON numbers, which unmarshal to
// float64. It returns false when want is nil or the observed value is absent
// or not numeric.
func DriftedInt(data map[string]interface{}, key string, want *int) bool {
	if want == nil {
		return false
	}
	got, ok := data[key].(float64)
	return ok && int(got) != *want
}

// DriftedStringSlice reports whether the observed Vault response value for key
// differs from want. It returns false when want is empty or the observed value
// is missing or not a list.
func DriftedStringSlice(data map[string]interface{}, key string, want []string) bool {
	if len(want) == 0 {
		return false
	}
	got, ok := data[key].([]interface{})
	return ok && !reflect.DeepEqual(StringSlice(got), want)
}

// DriftedStringMap reports whether the observed Vault response value for key
// differs from want. It returns false when want is empty or the observed value
// is missing or not an object.
func DriftedStringMap(data map[string]interface{}, key string, want map[string]string) bool {
	if len(want) == 0 {
		return false
	}
	got, ok := data[key].(map[string]interface{})
	return ok && !reflect.DeepEqual(StringMap(got), want)
}

// DriftedDuration reports whether the observed Vault response value for a
// duration key differs from want. Vault may return durations either as
// formatted strings (e.g. "1h") or as integer seconds, so both forms are
// normalised to seconds before comparison. It returns false when want is empty
// or the observed value cannot be parsed.
func DriftedDuration(data map[string]interface{}, key, want string) bool {
	if want == "" {
		return false
	}
	wantSec, err := durationSeconds(want)
	if err != nil {
		return false
	}
	switch got := data[key].(type) {
	case string:
		gotSec, err := durationSeconds(got)
		return err == nil && gotSec != wantSec
	case float64:
		return int64(got) != wantSec
	}
	return false
}

// DriftedIntDuration reports whether the observed Vault response value for a
// duration key differs from want seconds. It is used where the spec stores a
// duration as an integer number of seconds (e.g. role TTLs) while Vault may
// return it either as a formatted string (e.g. "15m") or as integer seconds.
// It returns false when want is nil or the observed value cannot be parsed.
func DriftedIntDuration(data map[string]interface{}, key string, want *int) bool {
	if want == nil {
		return false
	}
	switch got := data[key].(type) {
	case string:
		gotSec, err := durationSeconds(got)
		return err == nil && gotSec != int64(*want)
	case float64:
		return int64(got) != int64(*want)
	}
	return false
}

func durationSeconds(v string) (int64, error) {
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, err
	}
	return int64(d.Seconds()), nil
}
