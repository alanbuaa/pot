package utils

import (
	"fmt"
	"io"
	"os"
)

// AppendToFile appends content to a file. Creates the file if it doesn't exist.
// Returns error if operation fails.
func AppendToFile(filepath string, content string) error {
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filepath, err)
	}

	return nil
}

// WriteToFile writes content to a file, truncating it if it exists.
// Returns error if operation fails.
func WriteToFile(filepath string, content string) error {
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filepath, err)
	}

	return nil
}

// ReadFile reads the entire content of a file and returns it as a string.
// Returns error if file doesn't exist or can't be read.
func ReadFile(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filepath, err)
	}

	return string(content), nil
}

// ReadFileBytes reads the entire content of a file and returns it as bytes.
// Returns error if file doesn't exist or can't be read.
func ReadFileBytes(filepath string) ([]byte, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filepath, err)
	}

	return content, nil
}

// AppendLinesToFile appends multiple lines to a file with newline separators.
// Creates the file if it doesn't exist.
func AppendLinesToFile(filepath string, lines []string) error {
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	for _, line := range lines {
		_, err = file.WriteString(line + "\n")
		if err != nil {
			return fmt.Errorf("failed to write line to file %s: %w", filepath, err)
		}
	}

	return nil
}

// FileExists checks if a file exists at the given path.
func FileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}

// CreateFileIfNotExist creates an empty file if it doesn't exist.
// Returns error if creation fails.
func CreateFileIfNotExist(filepath string) error {
	if FileExists(filepath) {
		return nil
	}

	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filepath, err)
	}
	defer file.Close()

	return nil
}
