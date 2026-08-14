package step

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtifactSlugFor(t *testing.T) {
	tests := []struct {
		name     string
		urlMap   string
		fileName string
		want     string
	}{
		{
			name:     "single entry",
			urlMap:   "app-release.apk=>https://app.bitrise.io/artifact/abc123/download",
			fileName: "app-release.apk",
			want:     "abc123",
		},
		{
			name:     "picks the matching entry out of several",
			urlMap:   "Fruta iOS Clip.app.zip=>https://app.bitrise.io/artifact/clipslug/download|Fruta iOS.app.zip=>https://app.bitrise.io/artifact/appslug/download",
			fileName: "Fruta iOS.app.zip",
			want:     "appslug",
		},
		{
			name:     "matches a key that is a full path",
			urlMap:   "/bitrise/deploy/app-release.apk=>https://app.bitrise.io/artifact/abc123/download",
			fileName: "app-release.apk",
			want:     "abc123",
		},
		{
			name:     "no match",
			urlMap:   "other.apk=>https://app.bitrise.io/artifact/abc123/download",
			fileName: "app-release.apk",
			want:     "",
		},
		{
			name:     "empty map",
			urlMap:   "",
			fileName: "app-release.apk",
			want:     "",
		},
		{
			name:     "a customised map format we cannot parse is treated as a miss",
			urlMap:   "app-release.apk -> https://app.bitrise.io/artifact/abc123/download",
			fileName: "app-release.apk",
			want:     "",
		},
		{
			name:     "a URL without a download segment is treated as a miss",
			urlMap:   "app-release.apk=>https://app.bitrise.io/artifact/abc123",
			fileName: "app-release.apk",
			want:     "",
		},
		{
			name:     "does not confuse a suffix match for the whole name",
			urlMap:   "my-app-release.apk=>https://app.bitrise.io/artifact/abc123/download",
			fileName: "app-release.apk",
			want:     "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, ArtifactSlugFor(test.urlMap, test.fileName))
		})
	}
}
