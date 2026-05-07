package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 创建目录
func EnsureDirectory(directoryPath string) error {
	if err := os.MkdirAll(directoryPath, 0755); err != nil {
		return fmt.Errorf("create directory at %s: %w", directoryPath, err)
	}
	return nil
}

// 安全拼接文件路径
func SafeJoin(baseDirectoryPath string, userFileName string) (string, error) {
	if strings.TrimSpace(userFileName) == "" {
		return "", fmt.Errorf("file name is empty")
	}

	cleanUserFileName := filepath.Clean(userFileName)
	if filepath.IsAbs(cleanUserFileName) || cleanUserFileName == "." || strings.HasPrefix(cleanUserFileName, "..") {
		return "", fmt.Errorf("invalid file path: %s", userFileName)
	}

	absoluteBaseDirectoryPath, err := filepath.Abs(baseDirectoryPath)
	if err != nil {
		return "", fmt.Errorf("resolve base directory at %s: %w", baseDirectoryPath, err)
	}

	absoluteTargetFilePath, err := filepath.Abs(filepath.Join(absoluteBaseDirectoryPath, cleanUserFileName))
	if err != nil {
		return "", fmt.Errorf("resolve target file path for %s: %w", userFileName, err)
	}

	relativeTargetPath, err := filepath.Rel(absoluteBaseDirectoryPath, absoluteTargetFilePath)
	if err != nil {
		return "", fmt.Errorf("validate file path for %s: %w", userFileName, err)
	}
	if relativeTargetPath == ".." || strings.HasPrefix(relativeTargetPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid file path: %s", userFileName)
	}

	return absoluteTargetFilePath, nil
}

// 写入文件并创建父目录
func WriteFileCreatingDirectory(filePath string, fileContent []byte, fileMode os.FileMode) error {
	parentDirectoryPath := filepath.Dir(filePath)
	if err := EnsureDirectory(parentDirectoryPath); err != nil {
		return err
	}
	if err := os.WriteFile(filePath, fileContent, fileMode); err != nil {
		return fmt.Errorf("write file at %s: %w", filePath, err)
	}
	return nil
}

// 复制文件
func CopyFile(sourceFilePath string, targetFilePath string) error {
	sourceFileContent, err := os.ReadFile(sourceFilePath)
	if err != nil {
		return fmt.Errorf("read file at %s: %w", sourceFilePath, err)
	}
	return WriteFileCreatingDirectory(targetFilePath, sourceFileContent, 0644)
}
