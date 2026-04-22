package loadstrike

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
)

type runtimeConfigFile struct {
	LoadStrike runtimeConfigSection `json:"LoadStrike"`
}

type runtimeConfigSection struct {
	TestSuite                   string `json:"TestSuite"`
	TestName                    string `json:"TestName"`
	RestartIterationMaxAttempts *int   `json:"RestartIterationMaxAttempts"`
	DisableLicenseEnforcement   *bool  `json:"DisableLicenseEnforcement"`
}

// LoadConfig applies supported settings from a loadstrike JSON config file.
func (r *runnerState) LoadConfig(path string) *runnerState {
	if strings.TrimSpace(path) == "" {
		r.context.recordConfigError(errors.New("config path must be provided"))
		return r
	}
	config, err := loadRuntimeConfig(path)
	if err != nil {
		r.context.recordConfigError(err)
		return r
	}

	applyRuntimeConfig(&r.context, config)
	return r
}

// LoadInfraConfig records the infra-config file path for sinks and plugins.
func (r *runnerState) LoadInfraConfig(path string) *runnerState {
	r.context.LoadInfraConfig(path)
	return r
}

func loadRuntimeConfig(path string) (runtimeConfigFile, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return runtimeConfigFile{}, err
	}

	var config runtimeConfigFile
	if err := json.Unmarshal(body, &config); err != nil {
		return runtimeConfigFile{}, err
	}
	if config.LoadStrike.DisableLicenseEnforcement != nil {
		return runtimeConfigFile{}, errors.New("disable license enforcement has been removed and is no longer supported")
	}

	return config, nil
}

func applyRuntimeConfig(context *contextState, config runtimeConfigFile) {
	if context == nil {
		return
	}

	if config.LoadStrike.TestSuite != "" {
		context.TestSuite = config.LoadStrike.TestSuite
	}

	if config.LoadStrike.TestName != "" {
		context.TestName = config.LoadStrike.TestName
	}

	if config.LoadStrike.RestartIterationMaxAttempts != nil && *config.LoadStrike.RestartIterationMaxAttempts >= 0 {
		context.RestartIterationMaxAttempts = *config.LoadStrike.RestartIterationMaxAttempts
	}
}

func applyRunArgs(context *contextState, args []string) error {
	if context == nil {
		return nil
	}

	for _, arg := range args {
		name, value, ok := parseRunArg(arg)
		if !ok {
			continue
		}

		switch name {
		case "restartiterationmaxattempts":
			attempts, err := strconv.Atoi(value)
			if err == nil && attempts >= 0 {
				context.RestartIterationMaxAttempts = attempts
			}
		case "disablelicenseenforcement":
			return errors.New("disable license enforcement has been removed and is no longer supported")
		}
	}
	return nil
}

func parseRunArg(arg string) (string, string, bool) {
	if !strings.HasPrefix(arg, "--") {
		return "", "", false
	}

	name, value, found := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
	if !found || name == "" {
		return "", "", false
	}

	return strings.ToLower(strings.TrimSpace(name)), strings.TrimSpace(value), true
}
