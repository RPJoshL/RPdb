package language

// OsLanguage tries to get the language of the operating system
// as a two-digit code (ISO 639).
// If that failes the default specified language will be returned
func GetOsLanguage(def string) string {
	rtc := getLanguage()

	// If the os langauge could not be detected an empty string is returned
	if rtc == "" {
		return def
	} else {
		return rtc
	}
}
