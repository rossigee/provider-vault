package clients

import "testing"

func TestDriftedString(t *testing.T) {
	cases := []struct {
		name  string
		data  map[string]interface{}
		key   string
		want  string
		drift bool
	}{
		{name: "empty want is ignored", data: map[string]interface{}{"a": "x"}, key: "a", want: ""},
		{name: "equal", data: map[string]interface{}{"a": "x"}, key: "a", want: "x"},
		{name: "different", data: map[string]interface{}{"a": "x"}, key: "a", want: "y", drift: true},
		{name: "missing key ignored", data: map[string]interface{}{}, key: "a", want: "x"},
		{name: "wrong type ignored", data: map[string]interface{}{"a": 3}, key: "a", want: "x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DriftedString(tc.data, tc.key, tc.want); got != tc.drift {
				t.Errorf("DriftedString() = %v, want %v", got, tc.drift)
			}
		})
	}
}

func TestDriftedBool(t *testing.T) {
	tr, fl := true, false
	cases := []struct {
		name  string
		data  map[string]interface{}
		key   string
		want  *bool
		drift bool
	}{
		{name: "nil want is ignored", data: map[string]interface{}{"a": true}, key: "a"},
		{name: "equal", data: map[string]interface{}{"a": true}, key: "a", want: &tr},
		{name: "different", data: map[string]interface{}{"a": true}, key: "a", want: &fl, drift: true},
		{name: "missing key ignored", data: map[string]interface{}{}, key: "a", want: &tr},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DriftedBool(tc.data, tc.key, tc.want); got != tc.drift {
				t.Errorf("DriftedBool() = %v, want %v", got, tc.drift)
			}
		})
	}
}

func TestDriftedInt(t *testing.T) {
	five := 5
	cases := []struct {
		name  string
		data  map[string]interface{}
		key   string
		want  *int
		drift bool
	}{
		{name: "nil want is ignored", data: map[string]interface{}{"a": float64(5)}, key: "a"},
		{name: "equal", data: map[string]interface{}{"a": float64(5)}, key: "a", want: &five},
		{name: "different", data: map[string]interface{}{"a": float64(6)}, key: "a", want: &five, drift: true},
		{name: "missing key ignored", data: map[string]interface{}{}, key: "a", want: &five},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DriftedInt(tc.data, tc.key, tc.want); got != tc.drift {
				t.Errorf("DriftedInt() = %v, want %v", got, tc.drift)
			}
		})
	}
}

func TestDriftedStringSlice(t *testing.T) {
	cases := []struct {
		name  string
		data  map[string]interface{}
		key   string
		want  []string
		drift bool
	}{
		{name: "empty want is ignored", data: map[string]interface{}{"a": []interface{}{"x"}}, key: "a", want: []string{"x"}},
		{name: "equal", data: map[string]interface{}{"a": []interface{}{"x", "y"}}, key: "a", want: []string{"x", "y"}},
		{name: "different", data: map[string]interface{}{"a": []interface{}{"x"}}, key: "a", want: []string{"y"}, drift: true},
		{name: "missing key ignored", data: map[string]interface{}{}, key: "a", want: []string{"x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DriftedStringSlice(tc.data, tc.key, tc.want); got != tc.drift {
				t.Errorf("DriftedStringSlice() = %v, want %v", got, tc.drift)
			}
		})
	}
}

func TestDriftedStringMap(t *testing.T) {
	cases := []struct {
		name  string
		data  map[string]interface{}
		key   string
		want  map[string]string
		drift bool
	}{
		{name: "empty want is ignored", data: map[string]interface{}{"a": map[string]interface{}{"k": "v"}}, key: "a", want: map[string]string{}},
		{name: "equal", data: map[string]interface{}{"a": map[string]interface{}{"k": "v"}}, key: "a", want: map[string]string{"k": "v"}},
		{name: "different", data: map[string]interface{}{"a": map[string]interface{}{"k": "v"}}, key: "a", want: map[string]string{"k": "w"}, drift: true},
		{name: "missing key ignored", data: map[string]interface{}{}, key: "a", want: map[string]string{"k": "v"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DriftedStringMap(tc.data, tc.key, tc.want); got != tc.drift {
				t.Errorf("DriftedStringMap() = %v, want %v", got, tc.drift)
			}
		})
	}
}

func TestDriftedDuration(t *testing.T) {
	cases := []struct {
		name  string
		data  map[string]interface{}
		key   string
		want  string
		drift bool
	}{
		{name: "empty want is ignored", data: map[string]interface{}{"a": "1h"}, key: "a", want: ""},
		{name: "equal string", data: map[string]interface{}{"a": "1h"}, key: "a", want: "1h"},
		{name: "equivalent formats", data: map[string]interface{}{"a": "60m"}, key: "a", want: "1h"},
		{name: "integer seconds", data: map[string]interface{}{"a": float64(3600)}, key: "a", want: "1h"},
		{name: "different", data: map[string]interface{}{"a": "2h"}, key: "a", want: "1h", drift: true},
		{name: "missing key ignored", data: map[string]interface{}{}, key: "a", want: "1h"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DriftedDuration(tc.data, tc.key, tc.want); got != tc.drift {
				t.Errorf("DriftedDuration() = %v, want %v", got, tc.drift)
			}
		})
	}
}

func TestDriftedIntDuration(t *testing.T) {
	oneHour := 3600
	cases := []struct {
		name  string
		data  map[string]interface{}
		key   string
		want  *int
		drift bool
	}{
		{name: "nil want is ignored", data: map[string]interface{}{"a": "1h"}, key: "a"},
		{name: "equal string", data: map[string]interface{}{"a": "1h"}, key: "a", want: &oneHour},
		{name: "equal seconds", data: map[string]interface{}{"a": float64(3600)}, key: "a", want: &oneHour},
		{name: "different", data: map[string]interface{}{"a": "2h"}, key: "a", want: &oneHour, drift: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DriftedIntDuration(tc.data, tc.key, tc.want); got != tc.drift {
				t.Errorf("DriftedIntDuration() = %v, want %v", got, tc.drift)
			}
		})
	}
}
