package shortner

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var ErrInvalidURL = errors.New("invalid URL")

type URLValidator struct {
	MaxLength int
}

func NewURLValidator() URLValidator {
	return URLValidator{MaxLength: MaxURLLength}
}

func (validator URLValidator) Validate(source string) error {
	maxLength := validator.MaxLength
	if maxLength <= 0 {
		maxLength = MaxURLLength
	}
	if source == "" || len(source) > maxLength || strings.TrimSpace(source) != source || !utf8.ValidString(source) {
		return ErrInvalidURL
	}
	for _, character := range source {
		if unicode.IsControl(character) {
			return ErrInvalidURL
		}
	}

	parsed, err := url.ParseRequestURI(source)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		return ErrInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}
	if parsed.User != nil || parsed.Hostname() == "" {
		return ErrInvalidURL
	}
	if port := parsed.Port(); port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return ErrInvalidURL
		}
	}
	if _, err := url.Parse(fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)); err != nil {
		return ErrInvalidURL
	}
	return nil
}

func IsValid(source string) bool {
	return NewURLValidator().Validate(source) == nil
}
