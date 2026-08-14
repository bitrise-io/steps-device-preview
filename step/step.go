package step

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/log"
)

const (
	previewURLEnvKey       = "BITRISE_DEVICE_PREVIEW_URL"
	previewExpiresAtEnvKey = "BITRISE_DEVICE_PREVIEW_EXPIRES_AT"

	// RDE rejects anything above this instead of clamping it, so catch it locally where the
	// message can name the input.
	maxLinkTTLHours = 72
)

// Input is the raw Step configuration, as the Bitrise CLI provides it.
type Input struct {
	AppPath     string `env:"app_path,required"`
	Platform    string `env:"platform"`
	DeviceModel string `env:"device_model"`
	OSVersion   string `env:"os_version"`

	LinkTTLHours  string `env:"link_ttl_hours"`
	PostPRComment bool   `env:"post_pr_comment,opt[true,false]"`

	PermanentDownloadURLMap string `env:"permanent_download_url_map"`

	BuildURL      string          `env:"build_url,required"`
	BuildAPIToken stepconf.Secret `env:"build_api_token,required"`

	FailOnError bool `env:"fail_on_error,opt[true,false]"`
	Verbose     bool `env:"verbose,opt[true,false]"`
}

// Config is the validated Step configuration.
type Config struct {
	AppPath     string
	Platform    string
	DeviceModel string
	OSVersion   string

	LinkTTLSeconds int
	PostPRComment  bool

	PermanentDownloadURLMap string

	BuildURL      string
	BuildAPIToken string

	FailOnError bool
}

// Result is what the Step exports once a preview link exists.
type Result struct {
	PreviewURL string
	ExpiresAt  string
}

// DevicePreview creates a device preview link for an app built in this build.
type DevicePreview struct {
	logger      log.Logger
	inputParser stepconf.InputParser
}

// New ...
func New(logger log.Logger, inputParser stepconf.InputParser) DevicePreview {
	return DevicePreview{logger: logger, inputParser: inputParser}
}

// ProcessConfig ...
func (s DevicePreview) ProcessConfig() (Config, error) {
	var input Input
	if err := s.inputParser.Parse(&input); err != nil {
		return Config{}, err
	}

	stepconf.Print(input)
	s.logger.EnableDebugLog(input.Verbose)

	if input.Platform != "" && input.Platform != PlatformIOS && input.Platform != PlatformAndroid {
		return Config{}, fmt.Errorf("platform must be %q, %q or empty, got %q", PlatformIOS, PlatformAndroid, input.Platform)
	}

	if _, err := os.Stat(input.AppPath); err != nil {
		return Config{}, fmt.Errorf("app_path %s is not readable: %w", input.AppPath, err)
	}

	ttlSeconds, err := linkTTLSeconds(input.LinkTTLHours)
	if err != nil {
		return Config{}, err
	}

	return Config{
		AppPath:                 input.AppPath,
		Platform:                input.Platform,
		DeviceModel:             input.DeviceModel,
		OSVersion:               input.OSVersion,
		LinkTTLSeconds:          ttlSeconds,
		PostPRComment:           input.PostPRComment,
		PermanentDownloadURLMap: input.PermanentDownloadURLMap,
		BuildURL:                input.BuildURL,
		BuildAPIToken:           string(input.BuildAPIToken),
		FailOnError:             input.FailOnError,
	}, nil
}

// Run prepares the app, makes sure it exists as a build artifact, and mints the preview link.
func (s DevicePreview) Run(config Config) (Result, error) {
	artifact, err := s.prepareArtifact(config.AppPath, config.Platform)
	if err != nil {
		return Result{}, err
	}
	s.logger.Printf("Platform: %s", artifact.Platform)

	client := newAPIClient(config.BuildURL, config.BuildAPIToken, s.logger)

	slug := ArtifactSlugFor(config.PermanentDownloadURLMap, filepath.Base(artifact.Path))
	if slug != "" {
		s.logger.Donef("Found %s among this build's artifacts (%s).", filepath.Base(artifact.Path), slug)
	} else {
		s.logger.Warnf("%s was not deployed by an earlier Deploy to Bitrise.io Step, uploading it now.", filepath.Base(artifact.Path))
		s.logger.Warnf("Add a Deploy to Bitrise.io Step before this one to avoid uploading the same file twice.")

		slug, err = client.UploadArtifact(artifact.Path)
		if err != nil {
			return Result{}, fmt.Errorf("upload %s: %w", artifact.Path, err)
		}
		s.logger.Donef("Uploaded as artifact %s.", slug)
	}

	s.logger.Println()
	s.logger.Infof("Creating the device preview link")

	result, err := client.CreateDevicePreview(slug, previewOptions{
		Platform:      artifact.Platform,
		DeviceModel:   config.DeviceModel,
		OSVersion:     config.OSVersion,
		TTLSeconds:    config.LinkTTLSeconds,
		PostPRComment: config.PostPRComment,
	})
	if err != nil {
		return Result{}, err
	}

	s.logger.Donef("Device preview: %s", result.PreviewURL)
	if result.ExpiresAt != "" {
		s.logger.Printf("The link expires at %s.", result.ExpiresAt)
	}

	return result, nil
}

// ExportOutputs ...
func (s DevicePreview) ExportOutputs(result Result) error {
	outputs := map[string]string{
		previewURLEnvKey:       result.PreviewURL,
		previewExpiresAtEnvKey: result.ExpiresAt,
	}

	for key, value := range outputs {
		if value == "" {
			continue
		}
		if err := tools.ExportEnvironmentWithEnvman(key, value); err != nil {
			return fmt.Errorf("export %s: %w", key, err)
		}
		s.logger.Donef("Exported %s", key)
	}

	return nil
}

func linkTTLSeconds(rawHours string) (int, error) {
	if rawHours == "" {
		return 0, nil // 0 means "use the deployment default", which is 24h
	}

	hours, err := strconv.Atoi(rawHours)
	if err != nil {
		return 0, fmt.Errorf("link_ttl_hours must be a whole number of hours, got %q", rawHours)
	}
	if hours <= 0 {
		return 0, fmt.Errorf("link_ttl_hours must be positive, got %d", hours)
	}
	if hours > maxLinkTTLHours {
		return 0, fmt.Errorf("link_ttl_hours must be at most %d, got %d", maxLinkTTLHours, hours)
	}

	return hours * 3600, nil
}
