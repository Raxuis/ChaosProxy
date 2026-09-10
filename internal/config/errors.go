package config

import "errors"

// ErrRuleNotFound identifies a runtime rule toggle targeting an unknown name.
var ErrRuleNotFound = errors.New("rule not found")
