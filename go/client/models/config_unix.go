//go:build unix || (js && wasm) || wasip1

package models

import "os"

// getUsersConfigFile returns the default path for the configuration file
// of this application
func getUsersConfigFile() (string, error) {
	dirName, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return dirName + "/.config/RPJosh/RPdb-go/config.yaml", nil
}
