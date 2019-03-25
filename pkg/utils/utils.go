// This package holds the Utils functions
package utils

import "os"

//If the Key is present in the environment the value (which may be empty) is returned.
// Otherwise the default value is returned
func GetEnv(key, defaultVal string) string {
	val, exists := os.LookupEnv(key)
	if exists {
		return val
	}
	return defaultVal
}
