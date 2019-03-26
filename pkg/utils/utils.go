// Package utils holds a common set of Util functions
package utils

import "os"

// GetEnv returns the value (which may be empty) If the Key is present in the environment
// Otherwise the default value is returned
func GetEnv(key, defaultVal string) string {
	val, exists := os.LookupEnv(key)
	if exists {
		return val
	}
	return defaultVal
}
