package validator

import (
	"errors"
	"regexp"
	"strings"
)

var emailRe = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)

func Email(s string) error {
	if !emailRe.MatchString(s) {
		return errors.New("invalid email")
	}
	return nil
}

func MinLen(field, s string, n int) error {
	if len(strings.TrimSpace(s)) < n {
		return errors.New(field + " too short")
	}
	return nil
}

func NotEmpty(field, s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New(field + " is required")
	}
	return nil
}

func OneOf(field, value string, options ...string) error {
	for _, o := range options {
		if value == o {
			return nil
		}
	}
	return errors.New(field + " must be one of: " + strings.Join(options, ", "))
}
