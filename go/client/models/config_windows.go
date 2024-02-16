package models

import "os"

// getUsersConfigFile returns the default path for the configuration file
// of this application
func getUsersConfigFile() (string, error) {
	return os.Getenv("APPDATA") + "\\RPJosh\\RPdb-go\\config.yaml", nil
}
