package step

import (
	"net/url"
	"strings"
)

// ArtifactSlugFor finds the artifact slug for fileName in a BITRISE_PERMANENT_DOWNLOAD_URL_MAP
// value, as exported by the Deploy to Bitrise.io Step.
//
// The map is `<file>=><url>` pairs joined by `|`, where the keys are base names and the URLs look
// like `https://app.bitrise.io/artifact/<slug>/download`. Returns "" when the file is not there,
// which is the caller's cue to upload it.
//
// The map format is user-configurable through the Deploy Step's
// `permanent_download_url_map_format` input, so treat a parse miss as "not found" rather than an
// error — uploading a second copy beats failing the build.
func ArtifactSlugFor(urlMap, fileName string) string {
	if urlMap == "" || fileName == "" {
		return ""
	}

	for _, entry := range strings.Split(urlMap, "|") {
		name, rawURL, found := strings.Cut(entry, "=>")
		if !found {
			continue
		}

		name = strings.TrimSpace(name)
		// The Deploy Step keys the map on base names, but older versions used full paths.
		if name != fileName && !strings.HasSuffix(name, "/"+fileName) {
			continue
		}

		if slug := slugFromDownloadURL(strings.TrimSpace(rawURL)); slug != "" {
			return slug
		}
	}

	return ""
}

// slugFromDownloadURL pulls the slug out of `.../<slug>/download`.
func slugFromDownloadURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i, segment := range segments {
		if segment == "download" && i > 0 {
			return segments[i-1]
		}
	}

	return ""
}
