package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

var supportedExtensions = map[string]bool{
	".mp3":  true,
	".flac": true,
	".ogg":  true,
	".m4a":  true,
	".wav":  true,
	".wma":  true,
	".opus": true,
}

func IsSupported(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return supportedExtensions[ext]
}

func ScanDirectory(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip files we can't access
		}
		if !info.IsDir() && IsSupported(path) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func FilterSupported(paths []string) []string {
	var result []string
	for _, p := range paths {
		if IsSupported(p) {
			result = append(result, p)
		}
	}
	return result
}
