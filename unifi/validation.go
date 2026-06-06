package unifi

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	vd "github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

// ValidationError is a custom error type for validation errors.
type ValidationError struct {
	Root     error
	Messages map[string]string
}

// Error returns the error message with combined all validation error messages.
// Field keys are sorted so the output is deterministic across runs (the
// underlying Messages map has nondeterministic iteration order).
func (v *ValidationError) Error() string {
	// With no per-field messages (e.g. the non-struct/InvalidValidationError
	// fallback in Validate), surface the root cause instead of rendering an empty
	// "validation failed: \n" body that drops the real error.
	if len(v.Messages) == 0 {
		if v.Root != nil {
			return "validation failed: " + v.Root.Error()
		}
		return "validation failed"
	}

	err := "validation failed: \n"
	fields := make([]string, 0, len(v.Messages))
	for field := range v.Messages {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	var errSb24 strings.Builder
	for _, field := range fields {
		fmt.Fprintf(&errSb24, "%s: %s\n", field, v.Messages[field])
	}
	err += errSb24.String()
	return err
}

// Unwrap exposes the underlying validator error so callers can use
// errors.Is/errors.As to reach the wrapped vd.ValidationErrors (or whatever
// raw error Validate fell back to).
func (v *ValidationError) Unwrap() error {
	return v.Root
}

// Validator is the interface for the validator. Use it to validate structs. You can register structure-level validations
// with RegisterStructValidation.
type Validator interface {
	// Validate validates the given struct and returns an error if the struct is not valid.
	Validate(i any) error
	// RegisterStructValidation registers a structure-level validation function for a given struct type.
	RegisterStructValidation(fn vd.StructLevelFunc, i any)
	// RegisterTranslation registers a custom translation for a given tag.
	RegisterTranslation(tag string, registerFn vd.RegisterTranslationsFunc, translationFn vd.TranslationFunc) (err error)
	// RegisterCustomValidator registers a custom validator function with own tag and error message.
	RegisterCustomValidator(cv CustomValidator) error
}

type validator struct {
	validate *vd.Validate
	trans    ut.Translator
}

func (v *validator) Validate(i any) error {
	if err := v.validate.Struct(i); err != nil {
		var errs vd.ValidationErrors
		// Validate.Struct can also return an *InvalidValidationError (e.g. a nil
		// or non-struct argument); guard the type assertion so we never call
		// Translate on a nil errs slice and panic.
		if !errors.As(err, &errs) {
			return &ValidationError{Root: err}
		}
		messages := errs.Translate(v.trans)

		return &ValidationError{Root: err, Messages: messages}
	}
	return nil
}

func (v *validator) RegisterStructValidation(f vd.StructLevelFunc, s any) {
	v.validate.RegisterStructValidation(f, s)
}

func (v *validator) RegisterTranslation(tag string, registerFn vd.RegisterTranslationsFunc, translationFn vd.TranslationFunc) error {
	return v.validate.RegisterTranslation(tag, v.trans, registerFn, translationFn)
}

func (v *validator) RegisterCustomValidator(cv CustomValidator) error {
	var err error
	if err = v.validate.RegisterValidation(cv.tag, cv.fn, false); err != nil {
		return fmt.Errorf("failed to register custom validation '%s': %w", cv.tag, err)
	}
	err = v.RegisterTranslation(cv.tag, func(ut ut.Translator) error {
		return ut.Add(cv.tag, cv.messageText, true)
	}, func(ut ut.Translator, fe vd.FieldError) string {
		t, _ := ut.T(cv.tag, append([]string{fe.Field()}, cv.params...)...)
		return t
	})
	if err != nil {
		return fmt.Errorf("failed to register custom validation '%s' translation: %w", cv.tag, err)
	}
	return nil
}

// newValidator builds a *validator pre-registered with the package's built-in
// custom validators (customValidators). Any additional one-off validators passed
// as extra are registered on top of those — WITHOUT mutating the shared
// customValidators global, so a test can register a throwaway validator on its own
// instance and not leak it into every other newValidator() call.
func newValidator(extra ...CustomValidator) (*validator, error) {
	validate := vd.New(vd.WithRequiredStructEnabled())
	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)
	trans, _ := uni.GetTranslator(enLocale.Locale())
	err := en_translations.RegisterDefaultTranslations(validate, trans)
	if err != nil {
		return nil, err
	}

	v := &validator{
		validate: validate,
		trans:    trans,
	}

	for _, customValidator := range customValidators {
		if err = v.RegisterCustomValidator(customValidator); err != nil {
			return nil, err
		}
	}
	for _, customValidator := range extra {
		if err = v.RegisterCustomValidator(customValidator); err != nil {
			return nil, err
		}
	}
	return v, nil
}

type CustomValidator struct {
	tag         string
	fn          vd.Func
	messageText string
	params      []string
}

func NewCustomRegexValidator(tag string, regex string) CustomValidator {
	cv := &CustomValidator{
		tag:         tag,
		messageText: regexValidatorMessage,
		params:      []string{regex},
	}
	crv := CustomRegexValidator{
		CustomValidator: cv,
		regex:           lazyRegexCompile(regex),
	}
	crv.fn = func(fl vd.FieldLevel) bool {
		return crv.regex().MatchString(fl.Field().String())
	}
	return *crv.CustomValidator
}

type CustomRegexValidator struct {
	*CustomValidator

	regex func() *regexp.Regexp
}

var customValidators = []CustomValidator{
	NewCustomRegexValidator("w_regex", wRegexString),
	NewCustomRegexValidator("numeric_nonzero", `^[1-9][0-9]*$`),
}

func lazyRegexCompile(str string) func() *regexp.Regexp {
	var regex *regexp.Regexp
	var once sync.Once
	return func() *regexp.Regexp {
		once.Do(func() {
			regex = regexp.MustCompile(str)
		})
		return regex
	}
}

const (
	regexValidatorMessage = "{0} must comply with the regular expression pattern '{1}'"
	wRegexString          = `^[\w]+$`
)
