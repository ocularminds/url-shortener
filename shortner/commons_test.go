package shortner

import (
	"strconv"
	"strings"
	"testing"
)

//TestHelloName calls greetings.Hello with a name,
//checking for a valid return value
func TestGetRandom(t *testing.T) {
	result := GetRandom()
	size := len(strings.Split(result, ""))
	if size != 8 {
		t.Fail()
	}
}

func TestIsValid(t *testing.T) {
	input := "https://www.google.com/search"
	msg := "IsValid() return %s, but exepect %s"
	result := IsValid(input)
	if result == false {
		t.Fatalf(msg, strconv.FormatBool(result), strconv.FormatBool(true))
	}
}

func TestIsValidWithInvalidUrl(t *testing.T) {
	input := "htt//wes.xo"
	msg := "IsValid() return %s, but exepect %s"
	result := IsValid(input)
	if result == true {
		t.Fatalf(msg, strconv.FormatBool(result), strconv.FormatBool(false))
	}
}
func TestShorten(t *testing.T) {
	longURL := "http://github.com/ocularminds/urlshortner"
	result, err := Shorten(longURL)
	if err != nil {
		t.Errorf("Shorten() %s throws Unexpected error", err.Error())
	}
	if len(result) >= len(longURL) {
		t.Errorf("Shorten() %s expected shorter url for %s", result, longURL)
	}
}

func TestShortWithInvalidUrl(t *testing.T) {
	input := "htp:00.1"
	result, err := Shorten(input)
	if err == nil {
		t.Errorf("Shorten() expected invalid error but was %s", result)
	}
}

func TestShortWithEmptyUrl(t *testing.T) {
	input := ""
	result, err := Shorten(input)
	if err == nil {
		t.Error("Shorten() with empty name expects to throw error")
	}
	if result != "" {
		t.Fail()
	}
}
