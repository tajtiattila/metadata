Metadata
========

[![GoDoc](https://godoc.org/github.com/tajtiattila/metadata?status.svg)](https://godoc.org/github.com/tajtiattila/metadata)

Package metadata parses metadata in media files.

	go get github.com/tajtiattila/metadata

# Supported metadata formats

	import _ "github.com/tajtiattila/metadata/exif"
	import _ "github.com/tajtiattila/metadata/xmp"

# Supported container formats

	import _ "github.com/tajtiattila/metadata/jpeg"
	import _ "github.com/tajtiattila/metadata/png"
	import _ "github.com/tajtiattila/metadata/mp4"

# Supported attributes

 | attribute name   | type      | Exif tag[1]      | XMP node[2]           |
 | ---------------- | --------- | ---------------- | --------------------- |
 | DateTimeCreated  | time.Time | DateTimeCreated  | xmp:CreateDate        |
 | DateTimeOriginal | time.Time | DateTimeOriginal | exif:DateTimeOriginal |
 | Rating           | int       |                  | xmp:Rating            |
 | Orientation      | int       | Orientation      | exif:Orientation      |
 | GPSLatitude      | float64   | GPSLatitude      | exif:GPSLatitude      |
 | GPSLongitude     | float64   | GPSLongitude     | exif:GPSLongitude     |
 | GPSDateTime      | time.Time | GPSLongitude     | exif:GPSLongitude     |
 | Make             | string    | Make             | tiff:Make             |
 | Model            | string    | Model            | tiff:Model            |

[1] Exif tags listed are from the exiftag package.
    Times are usually local times (without valid time zone)

[2] XMP prefixes map to their respective namespaces:

	xmp:  "http://ns.adobe.com/xap/1.0/"
	exif: "http://ns.adobe.com/exif/1.0/"
	tiff: "http://ns.adobe.com/tiff/1.0/"
