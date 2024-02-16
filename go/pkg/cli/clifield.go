package cli

import (
	"reflect"
	"strconv"
	"strings"

	"git.rpjosh.de/RPJosh/go-logger"
)

type cliField[T any] struct {
	shortKey string
	longKey  string

	reflectValue       reflect.Value
	setter             reflect.Value
	structField        reflect.StructField
	completionFunction reflect.Value

	// Only for child
	defaultValue *string
	requiredWith []string
	required     bool
	requiredPos  int

	// Only for root
	isRoot bool
	// If the run is only for bash completion
	isCompletion          bool
	completionOptionCheck reflect.Value
	help                  reflect.Value
	chields               []cliField[any]
}

// Fetches all the "cli" fields recursively from the given struct.
// It returns all the fields in a tree structure.
func getFields(structure any, root any) []cliField[any] {
	if reflect.ValueOf(structure).Kind() != reflect.Ptr {
		logger.Error("No pointer to struct given")
		return make([]cliField[any], 0)
	}

	concreteStruct := reflect.ValueOf(structure).Elem()
	sructType := reflect.TypeOf(structure).Elem()

	var fields []cliField[interface{}]
	for i := 0; i < sructType.NumField(); i++ {
		structField := sructType.Field(i)

		// Get the values from the tag
		tag := structField.Tag.Get("cli")
		if tag == "" {
			// Not a relevent field for cli parsing
			continue
		}
		tags := getValuesFromTag(tag)

		cliField := cliField[any]{
			shortKey:     tags[1],
			longKey:      tags[0],
			reflectValue: concreteStruct.Field(i),
			structField:  structField,
		}

		if isStruct(&structField) && !isStandardStruct(&structField) {
			if structField.Type.Kind() == reflect.Pointer {
				cliField.reflectValue = cliField.reflectValue.Elem()
			}

			// No "hierarchie was given" -> process on the same level
			if cliField.longKey == "" {
				fields = append(fields,
					getFields(cliField.reflectValue.Addr().Interface(), root)...,
				)
			} else {
				cliField.chields = getFields(cliField.reflectValue.Addr().Interface(), root)
				setupChildFromTag(tags, &cliField, structure)
				cliField.setupRootField()
			}
		} else {
			setupChildFromTag(tags, &cliField, structure)
		}

		fields = append(fields, cliField)
	}

	return fields
}

// Checks if the given Struct is a default struct like
// time.Time.
// These structs are not handled as a "struct" with hierarchi
// (only as raw values like string or int)
func isStandardStruct(structField *reflect.StructField) bool {
	switch structField.Type.String() {
	case "time.Time":
		return true
	// Custom standard structs
	case "models.NullString", "models.NullInt":
		return true
	}
	return false
}

// Gets all the values from the tag
func getValuesFromTag(tag string) []string {
	tags := strings.Split(tag, ",")
	if len(tags) < 2 {
		logger.Fatal("The tag value has to be at least two values long: shortKey,longKey,defaultValue,+")
	}

	return tags
}

// Fills all information for the child based on the tag values
func setupChildFromTag(tags []string, field *cliField[any], structure any) {

	// Setup autocomplete function
	field.completionFunction = getCompletionFunction(field.structField, structure)

	if len(tags) >= 3 && tags[2] != "" {
		field.defaultValue = &tags[2]

		// replace ~~ with an empty value
		if *field.defaultValue == "~~" {
			defaultValue := ""
			field.defaultValue = &defaultValue
		}
	}

	if len(tags) >= 4 {
		for i := 3; i < len(tags); i++ {
			if tags[i] == "+" {
				field.required = true
			} else if tags[i] != "-" && len(tags[i]) == 1 {
				number, err := strconv.Atoi(tags[i])
				if err != nil {
					logger.Error("Failed to parse the tag value %s to an integer (positional argument)", tags[i])
				} else {
					field.requiredPos = number
				}
			} else if tags[i] != "-" {
				field.requiredWith = append(field.requiredWith, strings.TrimLeft(tags[i], "+"))
			}
		}
	}

	method := reflect.ValueOf(structure).MethodByName("Set" + convertToPascalCase(field.structField.Name))
	if method.IsValid() {
		field.setter = method
	}
}

// Checks if the given struct field is a struct.
// This can either be a struct type or a pointer to a struct
func isStruct(structField *reflect.StructField) bool {
	return structField.Type.Kind() == reflect.Struct ||
		(structField.Type.Kind() == reflect.Pointer && structField.Type.Elem().Kind() == reflect.Struct)
}

// Sets the required properties for a root field
func (field *cliField[T]) setupRootField() {
	field.isRoot = true
	field.completionOptionCheck = getCompletionOptionCheckFunction(field.reflectValue.Addr())
	field.setHelp()
	field.setRootSetter()
}

// Searches for a help method and stores them inside the field
func (field *cliField[T]) setHelp() {
	method := field.reflectValue.Addr().MethodByName("Help")
	if !method.IsValid() {
		return
	}

	// the function should have no params
	if method.Type().NumIn() != 0 {
		logger.Warning("Help() function for field %s should have no params!", field.reflectValue.Type().Name())
		return
	}

	field.help = method
}

// Searches for an SetRoot method and stores them inside the field
func (field *cliField[T]) setRootSetter() {
	method := field.reflectValue.Addr().MethodByName("Set" + convertToPascalCase(field.reflectValue.Type().Name()))
	if method.IsValid() {
		if method.Type().NumIn() == 0 || method.Type().NumIn() == 1 {
			field.setter = method
		} else {
			logger.Error("Expected no or one parameter (entry struct) for the method %s", "Set"+convertToPascalCase(field.reflectValue.Type().Name()))
		}
	}
}

// Converts the first letter of all words to upper case
func convertToPascalCase(text string) string {
	//return cases.Title(language.English, cases.Compact).String(text)
	if len(text) == 0 {
		return text
	}

	return strings.ToUpper(text[0:1]) + text[1:]
}
