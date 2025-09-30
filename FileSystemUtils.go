package ctrlxutils

import (
	"os"
	"path/filepath"

	"github.com/uoul/go-common/serialization"
)

func MarshalAppdata[T any](serializer serialization.ISerializer, file string, permission os.FileMode, data T) error {
	filePath := filepath.Join(getAppdataRoot(), file)
	basePath := filepath.Dir(filePath)

	// Create path if not exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		if err := os.MkdirAll(basePath, os.ModePerm); err != nil {
			return err
		}
	}
	// Marshal Data
	content, err := serializer.Marshal(data)
	if err != nil {
		return err
	}
	// Write file
	if err := os.WriteFile(filePath, content, permission); err != nil {
		return err
	}
	// Data written successfully
	return nil
}

func UnmarshalAppdata[T any](serializer serialization.ISerializer, file string) (T, error) {
	filePath := filepath.Join(getAppdataRoot(), file)
	// Check file stats (e.g. file exists)
	if _, err := os.Stat(filePath); err != nil {
		return *new(T), err
	}
	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return *new(T), err
	}
	// Unmarshal data
	data := *new(T)
	if err := serializer.Unmarshal(content, &data); err != nil {
		return *new(T), err
	}
	// Return
	return data, nil
}

func getAppdataRoot() string {
	return filepath.Join(os.Getenv("SNAP_COMMON"), "solutions", "activeConfiguration", filepath.Base(os.Args[0]))
}
