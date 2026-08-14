package main

import (
	"os"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/steps-device-preview/step"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := log.NewLogger()
	previewStep := createStep(logger)

	config, err := previewStep.ProcessConfig()
	if err != nil {
		// A bad config is a bug in the Workflow, so it fails the build regardless of
		// `fail_on_error` — that input is about the preview service, not about the yml.
		logger.Errorf("Failed to process Step inputs: %s", err)
		return 1
	}

	result, err := previewStep.Run(config)
	if err != nil {
		logger.Errorf("Failed to create the device preview: %s", err)
		if config.FailOnError {
			return 1
		}
		logger.Warnf("Not failing the build: the `fail_on_error` input is off.")
		return 0
	}

	if err := previewStep.ExportOutputs(result); err != nil {
		logger.Errorf("Failed to export Step outputs: %s", err)
		return 1
	}

	return 0
}

func createStep(logger log.Logger) step.DevicePreview {
	envRepository := env.NewRepository()
	inputParser := stepconf.NewInputParser(envRepository)

	return step.New(logger, inputParser)
}
