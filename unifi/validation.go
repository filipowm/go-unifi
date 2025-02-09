package unifi

import (
	"errors"
	"fmt"

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
func (v *ValidationError) Error() string {
	err := "validation failed: \n"
	for field, message := range v.Messages {
		err += fmt.Sprintf("%s: %s\n", field, message)
	}
	return err
}

// Validator is the interface for the validator. Use it to validate structs. You can register structure-level validations
// with RegisterStructValidation.
type Validator interface {
	// Validate validates the given struct and returns an error if the struct is not valid.
	Validate(i interface{}) error
	// RegisterStructValidation registers a structure-level validation function for a given struct type.
	RegisterStructValidation(fn vd.StructLevelFunc, i interface{})
	// RegisterTranslation registers a custom translation for a given tag.
	RegisterTranslation(tag string, registerFn vd.RegisterTranslationsFunc, translationFn vd.TranslationFunc) (err error)
}

type validator struct {
	validate *vd.Validate
	trans    ut.Translator
}

func (v *validator) Validate(i interface{}) error {
	if err := v.validate.Struct(i); err != nil {
		var errs vd.ValidationErrors
		errors.As(err, &errs)
		messages := errs.Translate(v.trans)

		return &ValidationError{Root: err, Messages: messages}
	}
	return nil
}

func (v *validator) RegisterStructValidation(f vd.StructLevelFunc, s interface{}) {
	v.validate.RegisterStructValidation(f, s)
}

func (v *validator) RegisterTranslation(tag string, registerFn vd.RegisterTranslationsFunc, translationFn vd.TranslationFunc) error {
	return v.validate.RegisterTranslation(tag, v.trans, registerFn, translationFn)
}

func newValidator() (*validator, error) {
	validate := vd.New(vd.WithRequiredStructEnabled())
	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)
	trans, _ := uni.GetTranslator(enLocale.Locale())
	err := en_translations.RegisterDefaultTranslations(validate, trans)
	if err != nil {
		return nil, err
	}

	return &validator{
		validate: validate,
		trans:    trans,
	}, nil
}
