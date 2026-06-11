package unifi

import (
	"fmt"
	"strconv"
	"strings"
)

func emptyBoolToTrue(b *bool) bool {
	if b == nil {
		return true
	}
	return *b
}

// numberOrString handles strings that can also accept JSON numbers.
// For example a field may contain a number or the string "auto".
type numberOrString string

func (e *numberOrString) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	s := strings.TrimSpace(string(b))
	// Treat the JSON null token like the empty-string token: a server-sent null
	// must NOT become the literal 4-character string "null", which would then be
	// marshaled straight back to the controller on the next PUT, corrupting the
	// value.
	if s == `""` || s == "null" {
		*e = ""
		return nil
	}
	// Quoted string: unquote and store the raw string value (e.g. "auto").
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		unquoted, err := strconv.Unquote(s)
		if err != nil {
			return err
		}
		*e = numberOrString(unquoted)
		return nil
	}
	// Bare JSON number: accept integers and floats, storing their textual form.
	// Reject any other unquoted scalar (true/false/objects/arrays) rather than
	// storing raw bytes that round-trip into a bogus value.
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return fmt.Errorf("numberOrString: cannot unmarshal %q as number or string", s)
	}
	*e = numberOrString(s)
	return nil
}

// emptyStringInt was created due to the behavior change in
// Go 1.14 with json.Number's handling of empty string.
type emptyStringInt int

func (e *emptyStringInt) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	s := string(b)
	if s == `""` || s == "null" {
		*e = 0
		return nil
	}
	var err error
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		s, err = strconv.Unquote(s)
		if err != nil {
			return err
		}
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*e = emptyStringInt(i)
	return nil
}

func (e *emptyStringInt) MarshalJSON() ([]byte, error) {
	if e == nil || *e == 0 {
		return []byte(`""`), nil
	}

	return []byte(strconv.Itoa(int(*e))), nil
}

type booleanishString bool

// UnmarshalJSON decodes the assorted truthy/falsy wire forms the controller has
// used for the same field over time. Historically it sent "enabled"/"disabled";
// current controllers (9.x–10.x) send bare JSON booleans true/false for the wired
// fields (Device.LtePoe/LteExtAnt). It is intentionally PERMISSIVE: it accepts
// bare and quoted booleans, "enabled"/"disabled", "1"/"0", and empty/null (→false),
// and NEVER hard-errors on an unrecognized scalar — a single bad field must not
// poison the whole Device decode. Unrecognized input decodes to false.
func (e *booleanishString) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	// Unquote string tokens so quoted and bare forms collapse to one switch.
	if len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		if unquoted, err := strconv.Unquote(s); err == nil {
			s = unquoted
		}
	}
	switch strings.ToLower(s) {
	case "true", "enabled", "1":
		*e = true
	default:
		// "false", "disabled", "0", "", "null", and anything unrecognized → false.
		*e = false
	}
	return nil
}
