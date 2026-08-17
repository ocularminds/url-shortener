package service

import "crypto/rand"

const slugAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

const DefaultSlugLength = 8

type SlugGenerator interface {
	Generate() (string, error)
}

type CryptoSlugGenerator struct {
	Length int
}

func (generator CryptoSlugGenerator) Generate() (string, error) {
	length := generator.Length
	if length <= 0 {
		length = DefaultSlugLength
	}

	result := make([]byte, length)
	randomBytes := make([]byte, length*2)
	for position := 0; position < length; {
		if _, err := rand.Read(randomBytes); err != nil {
			return "", err
		}
		for _, value := range randomBytes {
			// 248 is the largest multiple of 62 below 256. Discarding larger
			// values avoids modulo bias.
			if value >= 248 {
				continue
			}
			result[position] = slugAlphabet[int(value)%len(slugAlphabet)]
			position++
			if position == length {
				break
			}
		}
	}
	return string(result), nil
}
