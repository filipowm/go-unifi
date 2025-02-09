package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type validator string

type validation struct {
	v      validator
	params []string
}

type validationComment string

type regexSpecialChars string

const (
	validateTag           = "validate"
	required    validator = "required"
	mac         validator = "mac"
	ip          validator = "ip"
	ipv4        validator = "ipv4"
	ipv6        validator = "ipv6"
	httpUrl     validator = "http_url"
	oneOf       validator = "oneof"
	cidr        validator = "cidr"
	omitempty   validator = "omitempty"
	length      validator = "len"
	gte         validator = "gte"
	lte         validator = "lte"
	w_regex     validator = "w_regex"

	regexChars regexSpecialChars = "^$*+?()[]{}\\|."
)

func createValidations(validations ...validation) string {
	if len(validations) == 0 {
		return ""
	}
	validators := make([]string, len(validations)+1)
	validators[0] = createValidator(omitempty)
	for i, v := range validations {
		validators[i+1] = createValidator(v.v, v.params...)
	}
	joinedValidators := strings.Join(validators, ",")
	return fmt.Sprintf("%s:\"%s\"", validateTag, joinedValidators)
}

func createValidator(v validator, params ...string) string {
	var filteredParams []string
	for _, p := range params {
		if p != "" {
			filteredParams = append(filteredParams, p)
		}
	}
	if len(filteredParams) == 0 {
		return string(v)
	}
	return fmt.Sprintf("%s=%s", v, strings.Join(filteredParams, " "))
}

func (r regexSpecialChars) In(s string, excludedChars string) bool {
	for _, c := range r {
		if strings.ContainsRune(s, c) && !strings.ContainsRune(excludedChars, c) {
			return true
		}
	}
	return false
}

func (r regexSpecialChars) NotIn(s string, excludedChars string) bool {
	return !r.In(s, excludedChars)
}

func (vc validationComment) HasDefinedLength() bool {
	s := string(vc)
	formatOk := strings.HasPrefix(s, ".{") && strings.HasSuffix(s, "}") && regexChars.NotIn(s, ".{}")
	if formatOk {
		sub := s[2 : len(s)-1]
		bounds := strings.Split(sub, ",")
		if len(bounds) < 1 || len(bounds) > 2 {
			return false
		}
		for _, b := range bounds {
			if _, err := strconv.Atoi(b); err != nil {
				return false
			}
		}
		return true
	}
	return false
}

func (vc validationComment) IsOneOf() bool {
	s := string(vc)
	return strings.Contains(s, "|") && regexChars.NotIn(s, "|")
}

func (vc validationComment) IsWRegex() bool {
	s := string(vc)
	return slices.Contains([]string{"[\\d\\w]+", "[\\d\\w]*", "[\\w]+", "[\\w]*"}, s)
}

func defineFieldValidation(rawValidation string) string {
	if rawValidation == "" {
		return ""
	}
	vc := validationComment(rawValidation)
	if vc.IsOneOf() {
		return createValidations(validation{v: oneOf, params: strings.Split(rawValidation, "|")})
	} else if vc.HasDefinedLength() {
		sub := rawValidation[2 : len(rawValidation)-1]
		bounds := strings.Split(sub, ",")
		if len(bounds) == 1 {
			return createValidations(validation{v: length, params: []string{bounds[0]}})
		}
		return createValidations(validation{v: gte, params: []string{bounds[0]}}, validation{v: lte, params: []string{bounds[1]}})
	} else if vc.IsWRegex() {
		return createValidations(validation{v: w_regex})
	}
	return ""
}
