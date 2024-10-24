# Image Vault

Image Vault is a powerful command-line tool specifically designed for managing and organizing photo libraries. It provides photographers and photo enthusiasts with features for initializing photo libraries, adding and organizing image files, verifying library integrity, and displaying detailed photo metadata.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
  - [Initializing a Photo Library](#initializing-a-photo-library)
  - [Adding Photos](#adding-photos)
  - [Verifying Photo Library Integrity](#verifying-photo-library-integrity)
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

## Usage

Image Vault offers several commands to manage your photo library efficiently. Here are the main commands and their usage:

### Initializing a Photo Library

To initialize a new Image Vault photo library in the current directory:

```
imv init
```

This command creates a configuration file (`image-vault.yaml`) in the current directory, setting up the structure for your photo organization.

### Adding Photos

To add photos to your library:

```
imv add <path_to_photos>
```

This command processes the photos at the specified path, organizes them according to the configured template (which can include metadata like date taken, camera model, etc.), and adds them to the library.

### Verifying Photo Library Integrity

To verify the integrity of your photo library:

```
imv verify
```

This command checks all photos in the library to ensure they are properly organized and match their expected locations based on their metadata and the configured template.

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
template: |-
  {{- $make := or .Exif.Make .Exif.DeviceManufacturer "NoMake" -}}
  {{- $model := or .Exif.Model .Exif.DeviceModelName "NoModel" -}}
  {{- $dateTimeOriginal := and (any .Exif.DateTimeOriginal) (ne .Exif.DateTimeOriginal "0000:00:00 00:00:00") | ternary .Exif.DateTimeOriginal "" -}}
  {{- $mediaCreateDate := and (any .Exif.MediaCreateDate) (ne .Exif.MediaCreateDate "0000:00:00 00:00:00") | ternary .Exif.MediaCreateDate "" -}}
  {{- $date := or $dateTimeOriginal $mediaCreateDate "1970:01:01 00:00:00" | toDate "2006:01:02 15:04:05" -}}
  {{- $mimeType := .Exif.MIMEType | default "unknown/unknown" | splitList "/" | first -}}
  {{$make}} {{$model}} ({{$mimeType}})/{{$date | date "2006"}}/{{$date | date "2006-01-02"}}/{{$date | date "2006-01-02_15-04-05"}}_{{.Hash.Md5Short}}{{.Fs.Ext | lower}}

skipPermissionDenied: true

ignore:
  - image-vault.yaml
  - .*

sidecarExtensions:
  - "*.xmp"
  - "*.yaml"
  - "*.json"
```

You can modify this configuration to suit your photo organization needs. The `template` field is particularly important as it determines how your photos will be organized in the library based on their metadata.
