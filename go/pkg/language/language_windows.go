package language

import (
	"unsafe"

	"git.rpjosh.de/RPJosh/go-logger"
	"golang.org/x/sys/windows"
)

// Maximum result lenght of of GetUserDefaultLocaleName
const LocaleNameMaxLenght uint32 = 85
const FunctionName = "GetUserDefaultLocaleName"

// The language will be retreived through the GetUserDefaultLocaleName function
// provided by windows
func getLanguage() string {

	// Load windows dll
	dll, err := windows.LoadDLL("kernel32")
	if err != nil {
		logger.Debug("Failed to load kernel32 dll: %s", err)
		return ""
	}

	// Find function. This should be supported since Windows Vista
	proc, err := dll.FindProc(FunctionName)
	if err != nil {
		logger.Debug("Unable to find %q inside kernel32.dll. Are you running a windows < vista?", FunctionName)
		return ""
	}

	// Allocate memory for result and call function
	buffer := make([]uint16, LocaleNameMaxLenght)
	ret, _, _ := proc.Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(LocaleNameMaxLenght))
	if ret == 0 {
		logger.Debug("Locale was not found while calling %q: %s", FunctionName, err)
		return ""
	}

	result := windows.UTF16ToString(buffer)

	if len(result) < 2 {
		logger.Debug("Received invalid input from syscall %q: %q", FunctionName, result)
	} else {
		return result[0:2]
	}

	return ""
}
