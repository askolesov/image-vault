# Image Vault

[![Lint](https://github.com/askolesov/image-vault/actions/workflows/lint.yaml/badge.svg)](https://github.com/askolesov/image-vault/actions/workflows/lint.yaml)
[![Test](https://github.com/askolesov/image-vault/actions/workflows/test.yaml/badge.svg)](https://github.com/askolesov/image-vault/actions/workflows/test.yaml)
[![Version](https://img.shields.io/github/v/release/askolesov/image-vault?include_prereleases)](https://github.com/askolesov/image-vault/releases)

Image Vault is a command-line tool designed for organizing photo libraries in a deterministic way. It calculates the path for each photo based on its metadata and a configured template. Importing the same photo multiple times will always result in the same path, so duplicate files won't be added to the library. It's also possible to check library integrity (verifying all files are in place) and fix any inconsistencies.

## TLDR

```
mkdir Photos && cd Photos     # Create a new directory for your photos
imv init                      # Initialize a new photo library
imv import <path_to_photos>   # Import photos into the library
imv verify --fix              # Verify the library integrity and fix any issues
imv cleanup                   # Clean up empty directories in the library
```

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
  - [Initializing a Photo Library](#initializing-a-photo-library)
  - [Importing Photos](#importing-photos)
  - [Verifying Photo Library Integrity](#verifying-photo-library-integrity)
  - [Cleaning Up Empty Directories](#cleaning-up-empty-directories)
  - [Displaying Photo Metadata](#displaying-photo-metadata)
  - [Showing Version Information](#showing-version-information)
- [Configuration](#configuration)
- [Building from Source](#building-from-source)
- [Contributing](#contributing)
- [License](#license)

## Installation

To install Image Vault, use the following command:

```
go install github.com/askolesov/image-vault/cmd/imv@latest
```

Alternatively, you can build from source (see [Building from Source](#building-from-source)).

### Required Dependency

Image Vault requires `exiftool` to be installed on your system for handling photo metadata. Below are instructions for installing `exiftool` on different operating systems:

#### Linux

To install `exiftool` on Linux, you can use the package manager for your distribution. For example, on Ubuntu, you can use:

```
sudo apt install libimage-exiftool-perl
```

#### macOS

To install `exiftool` on macOS using Homebrew, first ensure you have Homebrew installed, then run:

```
brew install exiftool
```

#### Windows

To install `exiftool` on Windows, you can:

1. Download the Windows executable from the [official ExifTool website](https://exiftool.org/)
2. Or use Chocolatey package manager:
   ```
   choco install exiftool
   ```
3. Or use Scoop package manager:
   ```
   scoop install exiftool
   ```

## Usage

Image Vault offers several commands to manage your photo library efficiently. Here are the main commands and their usage:

### Initializing a Photo Library

To initialize a new Image Vault photo library in the current directory:

```
imv init
```

This command creates a configuration file (`image-vault.yaml`) in the current directory, setting up the structure for your photo organization.

### Importing Photos

To import photos into your library:

```
imv import <path_to_photos>
```

This command processes the photos at the specified path, organizes them according to the configured template (which can include metadata like date taken, camera model, etc.), and imports them into the library.

### Verifying Photo Library Integrity

To verify the integrity of your photo library:

```
imv verify
```

This command checks all photos in the library to ensure they are properly organized and match their expected locations based on their metadata and the configured template.

To verify and automatically fix any issues found:

```
imv verify --fix
```

### Cleaning Up Empty Directories

To clean up empty directories in your photo library:

```
imv cleanup
```

This command removes empty directories that may have been left behind after moving or reorganizing photos, helping to keep your library structure clean and organized.

### Displaying Photo Metadata

To show detailed metadata for a specific photo:

```
imv info <photo_path>
```

This command displays comprehensive metadata information for the specified photo, including camera settings, date taken, and other EXIF data.

### Showing Version Information

To display version information about Image Vault:

```
imv version
```

## Configuration

Image Vault uses a YAML configuration file (`image-vault.yaml`) to customize its behavior. Here's an example of the default configuration:

```yaml
# The template below forms the target path where images will be imported.
# It creates a structured directory based on image metadata and file information.

# template: Supports Go template syntax and Sprig functions.
# Available namespaces: fs, exif, hash
# For more details, run: image-vault info <file>

template: |-
  {{- $make := or .Exif.Make .Exif.DeviceManufacturer "NoMake" -}}
  {{- $model := or .Exif.Model .Exif.DeviceModelName "NoModel" -}}
  {{- $dateTimeOriginal := and (any .Exif.DateTimeOriginal) (ne .Exif.DateTimeOriginal "0000:00:00 00:00:00") | ternary .Exif.DateTimeOriginal "" -}}
  {{- $mediaCreateDate := and (any .Exif.MediaCreateDate) (ne .Exif.MediaCreateDate "0000:00:00 00:00:00") | ternary .Exif.MediaCreateDate "" -}}
  {{- $date := or $dateTimeOriginal $mediaCreateDate "1970:01:01 00:00:00" | toDate "2006:01:02 15:04:05" -}}
  {{- $mimeType := .Exif.MIMEType | default "unknown/unknown" | splitList "/" | first -}}
  {{$make}} {{$model}} ({{$mimeType}})/{{$date | date "2006"}}/{{$date | date "2006-01-02"}}/{{$date | date "2006-01-02_15-04-05"}}_{{.Hash.Md5Short}}{{.Fs.Ext | lower}}

# skipPermissionDenied: Controls whether to skip files and directories with permission denied errors.
skipPermissionDenied: true

# ignore: List of file paths to ignore (supports .gitignore patterns).
ignore:
  - "image-vault.yaml"
  - ".*"

# sidecarExtensions: List of file extensions for sidecar files.
sidecarExtensions:
  - ".xmp"
  - ".yaml"
  - ".json"

# ignoreFilesInCleanup: List of files to ignore when determining if a directory is empty during cleanup.
ignoreFilesInCleanup:
  - ".DS_Store"
```

You can modify this configuration to suit your photo organization needs. The `template` field is particularly important as it determines how your photos will be organized in the library based on their metadata.

## Building from Source

To build Image Vault from source:

1. Clone the repository:
   ```
   git clone https://github.com/askolesov/image-vault.git
   ```

2. Navigate to the project directory:
   ```
   cd image-vault
   ```

3. Build the project:
   ```
   make build
   ```

   This will create the `imv` binary in the `build/` directory.

4. (Optional) Install the binary:
   ```
   make install
   ```

## Contributing

Contributions to Image Vault are welcome! Please feel free to submit pull requests, create issues, or suggest improvements.

Before submitting a pull request, please ensure that:

1. Your code passes all tests:
   ```
   make test
   ```

2. Your code passes the linter:
   ```
   make lint
   ```

3. You've added tests for any new functionality.
