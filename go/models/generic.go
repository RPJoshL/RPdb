package models

type Formattable interface {

	// ToSlice transforms all "relevant" fields of the struct into a string slice
	// for csv
	ToSlice() []string

	// ToString returns "relevant" fields of the struct as a pretty string
	String() string
}
