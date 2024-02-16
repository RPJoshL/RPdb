//go:build unix

package language

import (
	"os"

	"git.rpjosh.de/RPJosh/go-logger"
)

func getLanguage() string {
	if lang, exists := os.LookupEnv("LANG"); !exists {
		logger.Debug("Unable to determine language. Environment variable 'LANG' not set")
		return ""
	} else {
		// The variable was found and should contain something like this: "de_DE.UTF-8"
		if len(lang) < 2 {
			logger.Debug("Received invalid input from env variable 'LANG': %q", lang)
		} else {
			return lang[0:2]
		}

		return ""
	}
}
