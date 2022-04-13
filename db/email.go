package db

import "regexp"

const (
	pID     = `([a-z]([a-z0-9_-]*[a-z0-9])*)`
	pDotIDs = pID + `(\.` + pID + `)*`
	pEmail  = pDotIDs + `@` + pDotIDs + `\.` + pID //not super strict but checks at least a@b.c
)

var emailRegex = regexp.MustCompile("^" + pEmail + "$")

func ValidEmail(s string) bool {
	return emailRegex.MatchString(s)
}
