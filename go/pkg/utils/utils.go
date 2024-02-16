// utils contains some generic helper futions that makes your
// life easier
package utils

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Remove Removes one element from the slice.
// The order of the elements won't be preserved for performance
func Remove[T any](s *[]T, i int) []T {
	(*s)[i] = (*s)[len(*s)-1]
	return (*s)[:len(*s)-1]
}

// Filter filters the elements of a and b by the given compare function.
// If 'true' is returned from the filter, the element is removed from booth
// a and b and added to the response.
// The order of the elements of b won't be preserved (for "a" it will be preserved).
func Filter[TA any, TB any](a *[]TA, b *[]TB, comp func(a TA, b TB) bool) ([]TA, []TB) {
	rtcA := make([]TA, 0)
	rtcB := make([]TB, 0)

	iA := 0
outer:
	for indexA, aE := range *a {

		for indexB, bE := range *b {
			if comp(aE, bE) {
				// Add element to return and remove from slice
				rtcA = append(rtcA, (*a)[indexA])
				rtcB = append(rtcB, (*b)[indexB])

				// The same "performant" operation like for a cannot be used for b
				*b = Remove(b, indexB)

				continue outer
			}
		}

		// The value should be kept because "continue" isn't called
		(*a)[iA] = aE
		iA++
	}
	// Cut off remaining values
	*a = (*a)[:iA]

	return rtcA, rtcB
}

// Sprintfl returns the given message formatted with the locale
// language (currently only German) for placeholder.
// See "fmt.Sprintf()" for formatting options
func Sprintfl(msg string, placeholder ...any) string {
	p := message.NewPrinter(language.German)
	return p.Sprintf(msg, placeholder...)
}
