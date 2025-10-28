package utils

import (
	"os"
	"testing"
)

func TestAppendToFile(t *testing.T) {
	testFile := "test_append.txt"
	defer os.Remove(testFile)

	content := "Test content\n"
	err := AppendToFile(testFile, content)
	if err != nil {
		t.Fatalf("AppendToFile failed: %v", err)
	}

	// Append more content
	err = AppendToFile(testFile, content)
	if err != nil {
		t.Fatalf("Second AppendToFile failed: %v", err)
	}

	// Read and verify
	result, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	expected := content + content
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWriteToFile(t *testing.T) {
	testFile := "test_write.txt"
	defer os.Remove(testFile)

	content1 := "First content\n"
	err := WriteToFile(testFile, content1)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	// Overwrite with new content
	content2 := "Second content\n"
	err = WriteToFile(testFile, content2)
	if err != nil {
		t.Fatalf("Second WriteToFile failed: %v", err)
	}

	// Read and verify
	result, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if result != content2 {
		t.Errorf("Expected %q, got %q", content2, result)
	}
}

func TestAppendLinesToFile(t *testing.T) {
	testFile := "test_lines.txt"
	defer os.Remove(testFile)

	lines := []string{"Line 1", "Line 2", "Line 3"}
	err := AppendLinesToFile(testFile, lines)
	if err != nil {
		t.Fatalf("AppendLinesToFile failed: %v", err)
	}

	result, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	expected := "Line 1\nLine 2\nLine 3\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestFileExists(t *testing.T) {
	testFile := "test_exists.txt"

	// Should not exist initially
	if FileExists(testFile) {
		t.Error("File should not exist initially")
	}

	// Create the file
	err := WriteToFile(testFile, "test")
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}
	defer os.Remove(testFile)

	// Should exist now
	if !FileExists(testFile) {
		t.Error("File should exist after creation")
	}
}

func TestReadFileBytes(t *testing.T) {
	testFile := "test_bytes.txt"
	defer os.Remove(testFile)

	content := []byte{0x01, 0x02, 0x03, 0x04}
	err := os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result, err := ReadFileBytes(testFile)
	if err != nil {
		t.Fatalf("ReadFileBytes failed: %v", err)
	}

	if len(result) != len(content) {
		t.Errorf("Expected %d bytes, got %d", len(content), len(result))
	}

	for i, b := range result {
		if b != content[i] {
			t.Errorf("Byte %d: expected %d, got %d", i, content[i], b)
		}
	}
}
