package metadata

const (
	// Date/time values, with optional time zone
	// but otherwise formatted as RFC3339.
	//
	// Note: Exif has no way to specify time zone,
	// GPS location can be used to deduce it. From Exif the corresponding
	// SubSecTime is included in the values reported.
	DateTimeOriginal = "DateTimeOriginal" // date of original image (eg. scanned photo)
	DateTimeCreated  = "DateTimeCreated"  // original file creation date (eg. time of scan)

	GPSDateTime = "GPSDateTime" // Date/time of GPS fix (RFC3339, always UTC)

	// latitude and longitude are signed floating point
	// values formatted with no exponent
	GPSLatitude  = "GPSLatitude"  // +north, -south
	GPSLongitude = "GPSLongitude" // +east, -west

	// Orientation (integer) 1..8, values same like exif
	Orientation = "Orientation"

	// XMP Rating (integer), -1: rejected, 0: unrated, 1..5: user rating
	Rating = "Rating"

	Make  = "Make"  // recording equipment manufacturer name
	Model = "Model" // recording equipment model name or number
)

type Metadata struct {
	// Attr lists metadata attributes
	Attr map[string]string
}
