package pathbuilder

import (
	"testing"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSourcePath(t *testing.T) {
	tests := []struct {
		name     string
		fm       *metadata.FileMetadata
		opts     Options
		expected string
	}{
		{
			name: "standard photo",
			fm: &metadata.FileMetadata{
				Make:      "Apple",
				Model:     "iPhone 15 Pro",
				DateTime:  time.Date(2024, 8, 20, 18, 45, 3, 0, time.UTC),
				MediaType: defaults.MediaTypePhoto,
				ShortHash: "a1b2c3d4",
				Extension: ".jpg",
			},
			opts:     Options{SeparateVideo: true},
			expected: "2024/sources/Apple iPhone 15 Pro (image)/2024-08-20/2024-08-20_18-45-03_a1b2c3d4.jpg",
		},
		{
			name: "video separate",
			fm: &metadata.FileMetadata{
				Make:      "Apple",
				Model:     "iPhone 15 Pro",
				DateTime:  time.Date(2024, 8, 20, 18, 45, 3, 0, time.UTC),
				MediaType: defaults.MediaTypeVideo,
				ShortHash: "d4e5f6a7",
				Extension: ".mp4",
			},
			opts:     Options{SeparateVideo: true},
			expected: "2024/sources/Apple iPhone 15 Pro (video)/2024-08-20/2024-08-20_18-45-03_d4e5f6a7.mp4",
		},
		{
			name: "video not separate",
			fm: &metadata.FileMetadata{
				Make:      "Apple",
				Model:     "iPhone 15 Pro",
				DateTime:  time.Date(2024, 8, 20, 18, 45, 3, 0, time.UTC),
				MediaType: defaults.MediaTypeVideo,
				ShortHash: "d4e5f6a7",
				Extension: ".mp4",
			},
			opts:     Options{SeparateVideo: false},
			expected: "2024/sources/Apple iPhone 15 Pro (image)/2024-08-20/2024-08-20_18-45-03_d4e5f6a7.mp4",
		},
		{
			name: "unknown make no model",
			fm: &metadata.FileMetadata{
				Make:      "Unknown",
				Model:     "",
				DateTime:  time.Date(2025, 3, 15, 10, 30, 0, 0, time.UTC),
				MediaType: defaults.MediaTypePhoto,
				ShortHash: "abcd1234",
				Extension: ".jpg",
			},
			opts:     Options{},
			expected: "2025/sources/Unknown (image)/2025-03-15/2025-03-15_10-30-00_abcd1234.jpg",
		},
		{
			name: "audio file",
			fm: &metadata.FileMetadata{
				Make:      "Zoom",
				Model:     "H6",
				DateTime:  time.Date(2024, 5, 10, 14, 20, 0, 0, time.UTC),
				MediaType: defaults.MediaTypeAudio,
				ShortHash: "ff001122",
				Extension: ".wav",
			},
			opts:     Options{SeparateVideo: true},
			expected: "2024/sources/Zoom H6 (audio)/2024-05-10/2024-05-10_14-20-00_ff001122.wav",
		},
		{
			name: "make with no model",
			fm: &metadata.FileMetadata{
				Make:      "Sony",
				Model:     "",
				DateTime:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				MediaType: defaults.MediaTypePhoto,
				ShortHash: "11223344",
				Extension: ".arw",
			},
			opts:     Options{},
			expected: "2024/sources/Sony (image)/2024-01-01/2024-01-01_00-00-00_11223344.arw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildSourcePath(tt.fm, tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildSidecarPath(t *testing.T) {
	result := BuildSidecarPath("2024/sources/Apple iPhone 15 Pro (image)/2024-08-20/2024-08-20_18-45-03_a1b2c3d4.jpg", ".xmp")
	assert.Equal(t, "2024/sources/Apple iPhone 15 Pro (image)/2024-08-20/2024-08-20_18-45-03_a1b2c3d4.xmp", result)
}

func TestBuildSourceFilename(t *testing.T) {
	dt := time.Date(2024, 8, 20, 18, 45, 3, 0, time.UTC)
	result := BuildSourceFilename(dt, "a1b2c3d4", ".jpg")
	assert.Equal(t, "2024-08-20_18-45-03_a1b2c3d4.jpg", result)
}

func TestDeviceDir(t *testing.T) {
	tests := []struct {
		name      string
		make_     string
		model     string
		mediaType defaults.MediaType
		expected  string
	}{
		{
			name:      "full make and model",
			make_:     "Apple",
			model:     "iPhone 15 Pro",
			mediaType: defaults.MediaTypePhoto,
			expected:  "Apple iPhone 15 Pro (image)",
		},
		{
			name:      "no model",
			make_:     "Sony",
			model:     "",
			mediaType: defaults.MediaTypePhoto,
			expected:  "Sony (image)",
		},
		{
			name:      "video type",
			make_:     "DJI",
			model:     "Mavic 3",
			mediaType: defaults.MediaTypeVideo,
			expected:  "DJI Mavic 3 (video)",
		},
		{
			name:      "audio type",
			make_:     "Zoom",
			model:     "H6",
			mediaType: defaults.MediaTypeAudio,
			expected:  "Zoom H6 (audio)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeviceDir(tt.make_, tt.model, tt.mediaType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateProcessedDirName(t *testing.T) {
	tests := []struct {
		name         string
		dirName      string
		expectedYear string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "valid",
			dirName:      "2024-08-20 Summer Vacation",
			expectedYear: "2024",
			wantErr:      false,
		},
		{
			name:         "valid with different year",
			dirName:      "2025-01-15 New Year Party",
			expectedYear: "2025",
			wantErr:      false,
		},
		{
			name:         "wrong year",
			dirName:      "2024-08-20 Summer Vacation",
			expectedYear: "2025",
			wantErr:      true,
			errContains:  "year",
		},
		{
			name:         "no event name",
			dirName:      "2024-08-20",
			expectedYear: "2024",
			wantErr:      true,
		},
		{
			name:         "double space",
			dirName:      "2024-08-20  Summer Vacation",
			expectedYear: "2024",
			wantErr:      true,
		},
		{
			name:         "no space",
			dirName:      "2024-08-20SummerVacation",
			expectedYear: "2024",
			wantErr:      true,
		},
		{
			name:         "invalid date",
			dirName:      "2024-13-40 Bad Date",
			expectedYear: "2024",
			wantErr:      true,
			errContains:  "date",
		},
		{
			name:         "empty",
			dirName:      "",
			expectedYear: "2024",
			wantErr:      true,
		},
		{
			name:         "just text",
			dirName:      "Summer Vacation",
			expectedYear: "2024",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProcessedDirName(tt.dirName, tt.expectedYear)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParseSourceFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
		wantDT   time.Time
		wantHash string
		wantExt  string
	}{
		{
			name:     "valid",
			filename: "2024-08-20_18-45-03_a1b2c3d4.jpg",
			wantErr:  false,
			wantDT:   time.Date(2024, 8, 20, 18, 45, 3, 0, time.UTC),
			wantHash: "a1b2c3d4",
			wantExt:  ".jpg",
		},
		{
			name:     "valid long hash",
			filename: "2024-08-20_18-45-03_a1b2c3d4e5f6a7b8.mp4",
			wantErr:  false,
			wantDT:   time.Date(2024, 8, 20, 18, 45, 3, 0, time.UTC),
			wantHash: "a1b2c3d4e5f6a7b8",
			wantExt:  ".mp4",
		},
		{
			name:     "invalid format",
			filename: "photo.jpg",
			wantErr:  true,
		},
		{
			name:     "no hash",
			filename: "2024-08-20_18-45-03.jpg",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSourceFilename(tt.filename)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantDT, result.DateTime)
				assert.Equal(t, tt.wantHash, result.Hash)
				assert.Equal(t, tt.wantExt, result.Ext)
			}
		})
	}
}
