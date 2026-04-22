package loadstrike

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type runLogger struct {
	path string
	file *os.File
}

func startRunLogger(context contextState, testInfo testInfo, nodeInfo nodeInfo) (*runLogger, error) {
	if hasCustomLoggerConfig(context.LoggerConfig) {
		return nil, nil
	}
	reportFolder, err := defaultArtifactFolder(context)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(reportFolder, 0o755); err != nil {
		return nil, err
	}
	fileName := defaultLogFileName(testInfo, nodeInfo)
	logPath := filepath.Join(reportFolder, fileName)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	logger := &runLogger{path: logPath, file: file}
	logger.writeLine("LoadStrike log started")
	return logger, nil
}

func (l *runLogger) pathValue() string {
	if l == nil {
		return ""
	}
	return l.path
}

func (l *runLogger) writeLine(message string) {
	if l == nil || l.file == nil {
		return
	}
	_, _ = fmt.Fprintf(l.file, "[%s] %s\n", time.Now().UTC().Format(time.RFC3339Nano), strings.TrimSpace(message))
}

func (l *runLogger) closeWithResult(result *runResult, runErr error) {
	if l == nil || l.file == nil {
		return
	}
	if runErr != nil {
		l.writeLine("Run failed: " + runErr.Error())
	} else if result != nil {
		l.writeLine(fmt.Sprintf("Run completed. requests=%d ok=%d fail=%d", result.AllRequestCount, result.AllOKCount, result.AllFailCount))
	} else {
		l.writeLine("Run completed.")
	}
	_ = l.file.Close()
	l.file = nil
}

func (l *runLogger) publicLogger() *LoadStrikeLogger {
	return newLoadStrikeLogger(func(level string, message string) {
		if l == nil {
			return
		}
		l.writeLine("[" + strings.TrimSpace(level) + "] " + strings.TrimSpace(message))
	})
}

func hasCustomLoggerConfig(config LoggerConfiguration) bool {
	if config == nil {
		return false
	}
	return len(config) > 0
}

func defaultArtifactFolder(context contextState) (string, error) {
	reportFolder := strings.TrimSpace(context.ReportFolder)
	if reportFolder != "" {
		return reportFolder, nil
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return ".", err
	}
	return filepath.Join(workingDirectory, "reports"), nil
}

func defaultLogFileName(testInfo testInfo, nodeInfo nodeInfo) string {
	timestamp := artifactTimestamp(testInfo.CreatedUTC)
	suffixParts := []string{}

	if nodeInfo.NodeType != NodeTypeSingleNode {
		if nodeInfo.NodeType == NodeTypeCoordinator {
			suffixParts = append(suffixParts, "coordinator")
		} else {
			suffixParts = append(suffixParts, "agent")
		}
	}

	currentMachine := strings.TrimSpace(currentMachineName())
	if machineName := strings.TrimSpace(nodeInfo.MachineName); machineName != "" &&
		(nodeInfo.NodeType != NodeTypeSingleNode || !strings.EqualFold(machineName, currentMachine)) {
		suffixParts = append(suffixParts, machineName)
	}

	suffix := ""
	if len(suffixParts) > 0 {
		sanitized := make([]string, 0, len(suffixParts))
		for _, value := range suffixParts {
			sanitized = append(sanitized, sanitizeReportFileName(value))
		}
		suffix = "-" + strings.Join(sanitized, "-")
	}

	return sanitizeReportFileName(fmt.Sprintf("loadstrike-log-%s%s.txt", timestamp, suffix))
}

func artifactTimestamp(value time.Time) string {
	if value.IsZero() {
		value = time.Now().UTC()
	}
	return value.UTC().Format("20060102_150405")
}
