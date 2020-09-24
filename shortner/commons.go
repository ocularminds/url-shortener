package shortner

import (
	"errors"
	"math/rand"
	"net/url"
	"strings"
	"time"
)

//set initial values
func init() {
	rand.Seed(time.Now().UnixNano())
}

//Shorten shortens provided target url
func Shorten(target string) (string, error) {
	if !IsValid(target) {
		return "", errors.New("Invalid url")
	}
	return GetRandom(), nil
}

//IsValid validates a url
func IsValid(source string) bool {
	r, err := url.Parse(source)
	return err == nil && r.Host != "" && r.Scheme != ""
}

//GetRandom generates random short string
func GetRandom() string {
	text := ""
	var CHARS = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	characters := strings.Split(CHARS, "")
	size := len(characters)
	for i := 0; i < 8; i++ {
		text += characters[rand.Intn(size)]
	}
	return text
}
