// Package homedir is used for expanding homedir in relative paths
package homedir

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// Expand expands a path that begins with "~" to the user's home directory.
// If the path doesn't start with "~", it returns the original path.
// Returns an error if the home directory cannot be determined.
func Expand(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}

	if len(path) > 1 && path[1] == '/' {
		// ~/, just replace ~ with home dir
		home, err := getHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}

	// ~/ not found, but still starts with ~ - could be ~otheruser (not supported here)
	return "", errors.New("cannot expand ~ for other users")
}

// getHomeDir returns the user's home directory using OS-specific methods.
func getHomeDir() (string, error) {
	// Try OS-specific environment variables first
	switch runtime.GOOS {
	case "windows":
		// Windows: HOMEDRIVE + HOMEPATH or USERPROFILE
		homeDrive := os.Getenv("HOMEDRIVE")
		homePath := os.Getenv("HOMEPATH")
		if homeDrive != "" && homePath != "" {
			return filepath.Join(homeDrive, homePath), nil
		}
		userProfile := os.Getenv("USERPROFILE")
		if userProfile != "" {
			return userProfile, nil
		}
	default:
		// Unix-like: HOME environment variable
		home := os.Getenv("HOME")
		if home != "" {
			return home, nil
		}
	}

	// Fallback: use os.UserHomeDir() (available since Go 1.12)
	home, err := os.UserHomeDir()
	if err == nil {
		return home, nil
	}

	// Ultimate fallback: try common paths (rare, but possible in restricted environments)
	if runtime.GOOS == "darwin" {
		// macOS: /Users/<username>
		username := os.Getenv("USER")
		if username != "" {
			return "/Users/" + username, nil
		}
	} else if runtime.GOOS == "linux" {
		// Linux: /home/<username>
		username := os.Getenv("USER")
		if username != "" {
			return "/home/" + username, nil
		}
	}

	return "", errors.New("could not determine home directory")
}
