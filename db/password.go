package db

import (
	"math/rand"
	"strings"

	"github.com/go-msvc/errors"
)

const (
	charsLower   = "abcdefghijklmnopqrstuvwxyz"
	charsUpper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	charsDigits  = "1234567890"
	charsSymbols = "!@#$%^&*()_+-={}[]:\";'\\|<>,.?/"
	charsAll     = charsLower + charsUpper + charsDigits + charsSymbols
)

var (
	charSets = map[string]string{
		"lowercase": charsLower,
		"uppercase": charsUpper,
		"digits":    charsDigits,
		"symbols":   charsSymbols,
	}
)

func newRandomPassword(n int) string {
	if n < len(charSets) {
		n = len(charSets)
	}
	s := ""

	//start with one char from each set at least
	for _, c := range charSets {
		s += string(c[rand.Intn(len(c))])
	}

	//add more chars from any set to fill the required length
	for len(s) < n {
		s += string(charsAll[rand.Intn(len(charsAll))])
	}

	sRune := []rune(s)
	rand.Shuffle(len(sRune), func(i, j int) {
		sRune[i], sRune[j] = sRune[j], sRune[i]
	})
	return string(sRune)
}

func CheckPasswordStrength(s string, minLen int) error {
	if len(s) < minLen {
		return errors.Errorf("shorter than %d characters", minLen)
	}
	count := map[string]int{}
	for n := range charSets {
		count[n] = 0
	}
	for _, c := range s {
		for n, cs := range charSets {
			if strings.ContainsRune(cs, c) {
				count[n]++
			}
		}
	}
	for n, c := range count {
		if c < 1 {
			return errors.Errorf("does not have any %s (%s)", n, charSets[n])
		}
	}
	return nil //password is strong enough
} //CheckPasswordStrength()
