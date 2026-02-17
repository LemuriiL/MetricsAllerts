package cli

import (
	"os"
	"strconv"
	"strings"
)

type StringFlag struct {
	Val   string
	IsSet bool
}

func (s *StringFlag) String() string { return s.Val }
func (s *StringFlag) Set(v string) error {
	s.Val = v
	s.IsSet = true
	return nil
}

type IntFlag struct {
	Val   int
	IsSet bool
}

func (i *IntFlag) String() string { return strconv.Itoa(i.Val) }
func (i *IntFlag) Set(v string) error {
	n, err := strconv.Atoi(v)
	if err != nil {
		return err
	}
	i.Val = n
	i.IsSet = true
	return nil
}

type BoolFlag struct {
	Val   bool
	IsSet bool
}

func (b *BoolFlag) String() string {
	if b.Val {
		return "true"
	}
	return "false"
}

func (b *BoolFlag) Set(v string) error {
	x, err := strconv.ParseBool(v)
	if err != nil {
		return err
	}
	b.Val = x
	b.IsSet = true
	return nil
}

func EnvString(key string) (string, bool) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	return v, true
}

func EnvInt(key string) (int, bool) {
	v, ok := EnvString(key)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

func EnvBool(key string) (bool, bool) {
	v, ok := EnvString(key)
	if !ok {
		return false, false
	}
	x, err := strconv.ParseBool(v)
	if err != nil {
		return false, false
	}
	return x, true
}

func PickString(envKey string, flagVal string, flagIsSet bool, def string) string {
	if v, ok := EnvString(envKey); ok {
		return v
	}
	if flagIsSet {
		return flagVal
	}
	return def
}

func PickInt(envKey string, flagVal int, flagIsSet bool, def int) int {
	if v, ok := EnvInt(envKey); ok {
		return v
	}
	if flagIsSet {
		return flagVal
	}
	return def
}

func PickBool(envKey string, flagVal bool, flagIsSet bool, def bool) bool {
	if v, ok := EnvBool(envKey); ok {
		return v
	}
	if flagIsSet {
		return flagVal
	}
	return def
}

func NormalizeKey(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.EqualFold(v, "none") {
		return ""
	}
	return v
}

func NormalizePositiveInt(v int, def int) int {
	if v <= 0 {
		return def
	}
	return v
}
