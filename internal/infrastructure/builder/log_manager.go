package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"snapdeploy-core/internal/domain/deployment"
)

// LogManager handles writing and managing deployment logs
type LogManager struct {
	baseDir string
}

// NewLogManager creates a new log manager
func NewLogManager(baseDir string) *LogManager {
	return &LogManager{
		baseDir: baseDir,
	}
}

// GetLogFilePath generates the log file path for a deployment
func (lm *LogManager) GetLogFilePath(deploymentID deployment.DeploymentID, createdAt time.Time) string {
	timestamp := createdAt.Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.log", deploymentID.String(), timestamp)
	return filepath.Join(lm.baseDir, filename)
}

// EnsureLogDirectory ensures the log directory exists
func (lm *LogManager) EnsureLogDirectory() error {
	if err := os.MkdirAll(lm.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	return nil
}

// WriteLog writes a log line to the deployment log file
func (lm *LogManager) WriteLog(logPath, line string) error {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] %s\n", timestamp, line)

	if _, err := f.WriteString(logLine); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}

// ReadLog reads the entire log file
func (lm *LogManager) ReadLog(logPath string) (string, error) {
	data, err := os.ReadFile(logPath)
	if err != nil {
		return "", fmt.Errorf("failed to read log file: %w", err)
	}
	return string(data), nil
}

// DeleteLog deletes a log file
func (lm *LogManager) DeleteLog(logPath string) error {
	if err := os.Remove(logPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete log file: %w", err)
	}
	return nil
}

