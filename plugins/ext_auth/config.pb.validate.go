// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: plugins/ext_auth/config.proto

package ext_auth

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"

	v1 "mosn.io/moe/api/v1"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
	_ = sort.Sort

	_ = v1.StatusCode(0)
)

// Validate checks the field values on Config with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Config) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Config with the rules defined in the
// proto definition for this message. If any rules are violated, the result is
// a list of violation errors wrapped in ConfigMultiError, or nil if none found.
func (m *Config) ValidateAll() error {
	return m.validate(true)
}

func (m *Config) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	oneofServicesPresent := false
	switch v := m.Services.(type) {
	case *Config_HttpService:
		if v == nil {
			err := ConfigValidationError{
				field:  "Services",
				reason: "oneof value cannot be a typed-nil",
			}
			if !all {
				return err
			}
			errors = append(errors, err)
		}
		oneofServicesPresent = true

		if all {
			switch v := interface{}(m.GetHttpService()).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, ConfigValidationError{
						field:  "HttpService",
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, ConfigValidationError{
						field:  "HttpService",
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(m.GetHttpService()).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return ConfigValidationError{
					field:  "HttpService",
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	default:
		_ = v // ensures v is used
	}
	if !oneofServicesPresent {
		err := ConfigValidationError{
			field:  "Services",
			reason: "value is required",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return ConfigMultiError(errors)
	}

	return nil
}

// ConfigMultiError is an error wrapping multiple validation errors returned by
// Config.ValidateAll() if the designated constraints aren't met.
type ConfigMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ConfigMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ConfigMultiError) AllErrors() []error { return m }

// ConfigValidationError is the validation error returned by Config.Validate if
// the designated constraints aren't met.
type ConfigValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ConfigValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ConfigValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ConfigValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ConfigValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ConfigValidationError) ErrorName() string { return "ConfigValidationError" }

// Error satisfies the builtin error interface
func (e ConfigValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sConfig.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ConfigValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ConfigValidationError{}

// Validate checks the field values on HttpService with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *HttpService) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on HttpService with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in HttpServiceMultiError, or
// nil if none found.
func (m *HttpService) ValidateAll() error {
	return m.validate(true)
}

func (m *HttpService) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if uri, err := url.Parse(m.GetUrl()); err != nil {
		err = HttpServiceValidationError{
			field:  "Url",
			reason: "value must be a valid URI",
			cause:  err,
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	} else if !uri.IsAbs() {
		err := HttpServiceValidationError{
			field:  "Url",
			reason: "value must be absolute",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if all {
		switch v := interface{}(m.GetTimeout()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, HttpServiceValidationError{
					field:  "Timeout",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, HttpServiceValidationError{
					field:  "Timeout",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetTimeout()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return HttpServiceValidationError{
				field:  "Timeout",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetAuthorizationRequest()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, HttpServiceValidationError{
					field:  "AuthorizationRequest",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, HttpServiceValidationError{
					field:  "AuthorizationRequest",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetAuthorizationRequest()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return HttpServiceValidationError{
				field:  "AuthorizationRequest",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetAuthorizationResponse()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, HttpServiceValidationError{
					field:  "AuthorizationResponse",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, HttpServiceValidationError{
					field:  "AuthorizationResponse",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetAuthorizationResponse()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return HttpServiceValidationError{
				field:  "AuthorizationResponse",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	// no validation rules for StatusOnError

	if len(errors) > 0 {
		return HttpServiceMultiError(errors)
	}

	return nil
}

// HttpServiceMultiError is an error wrapping multiple validation errors
// returned by HttpService.ValidateAll() if the designated constraints aren't met.
type HttpServiceMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m HttpServiceMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m HttpServiceMultiError) AllErrors() []error { return m }

// HttpServiceValidationError is the validation error returned by
// HttpService.Validate if the designated constraints aren't met.
type HttpServiceValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e HttpServiceValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e HttpServiceValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e HttpServiceValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e HttpServiceValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e HttpServiceValidationError) ErrorName() string { return "HttpServiceValidationError" }

// Error satisfies the builtin error interface
func (e HttpServiceValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sHttpService.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = HttpServiceValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = HttpServiceValidationError{}

// Validate checks the field values on AuthorizationRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *AuthorizationRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on AuthorizationRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// AuthorizationRequestMultiError, or nil if none found.
func (m *AuthorizationRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *AuthorizationRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	for idx, item := range m.GetHeadersToAdd() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, AuthorizationRequestValidationError{
						field:  fmt.Sprintf("HeadersToAdd[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, AuthorizationRequestValidationError{
						field:  fmt.Sprintf("HeadersToAdd[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return AuthorizationRequestValidationError{
					field:  fmt.Sprintf("HeadersToAdd[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return AuthorizationRequestMultiError(errors)
	}

	return nil
}

// AuthorizationRequestMultiError is an error wrapping multiple validation
// errors returned by AuthorizationRequest.ValidateAll() if the designated
// constraints aren't met.
type AuthorizationRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m AuthorizationRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m AuthorizationRequestMultiError) AllErrors() []error { return m }

// AuthorizationRequestValidationError is the validation error returned by
// AuthorizationRequest.Validate if the designated constraints aren't met.
type AuthorizationRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e AuthorizationRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e AuthorizationRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e AuthorizationRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e AuthorizationRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e AuthorizationRequestValidationError) ErrorName() string {
	return "AuthorizationRequestValidationError"
}

// Error satisfies the builtin error interface
func (e AuthorizationRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sAuthorizationRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = AuthorizationRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = AuthorizationRequestValidationError{}

// Validate checks the field values on AuthorizationResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *AuthorizationResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on AuthorizationResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// AuthorizationResponseMultiError, or nil if none found.
func (m *AuthorizationResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *AuthorizationResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return AuthorizationResponseMultiError(errors)
	}

	return nil
}

// AuthorizationResponseMultiError is an error wrapping multiple validation
// errors returned by AuthorizationResponse.ValidateAll() if the designated
// constraints aren't met.
type AuthorizationResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m AuthorizationResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m AuthorizationResponseMultiError) AllErrors() []error { return m }

// AuthorizationResponseValidationError is the validation error returned by
// AuthorizationResponse.Validate if the designated constraints aren't met.
type AuthorizationResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e AuthorizationResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e AuthorizationResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e AuthorizationResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e AuthorizationResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e AuthorizationResponseValidationError) ErrorName() string {
	return "AuthorizationResponseValidationError"
}

// Error satisfies the builtin error interface
func (e AuthorizationResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sAuthorizationResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = AuthorizationResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = AuthorizationResponseValidationError{}
