package mp4

import "time"

type TKHD struct {
	Version      byte
	Flags        [3]byte
	DateCreated  time.Time
	DateModified time.Time

	TrackId         uint32
	DurationInUnits uint64 // time length (in time units; see MVHD)

	Width, Height uint32 // fixed point, see FrameSize
}

var ErrShortTKHD = formatError("TKHD too short")

func DecodeTKHD(p []byte) (*TKHD, error) {
	h := new(TKHD)

	bp := newBoxParse(p)

	var err error
	h.Version, h.Flags, err = bp.versionFlags()
	if err != nil {
		return nil, err
	}

	h.DateCreated = bp.Date()
	h.DateModified = bp.Date()
	h.TrackId = bp.Uint32()
	bp.Skip(8)
	h.DurationInUnits = bp.UintVar()
	bp.Skip(48)
	h.Width = bp.Uint32()
	h.Height = bp.Uint32()

	if bp.Short() {
		return nil, ErrShortTKHD
	}

	return h, nil
}

func (t *TKHD) FrameSize() (w, h int) {
	return int(t.Width >> 16), int(t.Height >> 16)
}

/* TKHD http://xhelmboyx.tripod.com/formats/mp4-layout.txt

* 8+ bytes track (element) box = long unsigned offset + long ASCII text string 'trak'

     * 8+ bytes track (element) header box
         = long unsigned offset + long ASCII text string 'tkhd'
       -> 1 byte version = byte unsigned value
         - if version is 1 then date and duration values are 8 bytes in length
       -> 3 bytes flags = 24-bit unsigned flags
         - sum of TrackEnabled = 1 ; TrackInMovie = 2 ;
            TrackInPreview = 4; TrackInPoster = 8
         - MPEG-4 only defines TrackEnabled as being valid

       -> 4 bytes created mac UTC date
           = long unsigned value in seconds since beginning 1904 to 2040
       -> 4 bytes modified mac UTC date
           = long unsigned value in seconds since beginning 1904 to 2040
       OR
       -> 8 bytes created mac UTC date
           = 64-bit unsigned value in seconds since beginning 1904
       -> 8 bytes modified mac UTC date
           = 64-bit unsigned value in seconds since beginning 1904

       -> 4 bytes track id = long integer value (first track = 1)
       -> 8 bytes reserved = 2 * long value set to zero

       -> 4 bytes duration = long unsigned time length (in time units)
       OR
       -> 8 bytes duration = 64-bit unsigned time length (in time units)
         - if duration is undefined set above bits to all ones

       -> 4 bytes reserved = long value set to zero
       -> 2 bytes video layer = short integer positon
           (middle = 0 ; negatives are in front)
       -> 2 bytes QUICKTIME alternate/other = short integer track id (none = 0)
       -> 2 bytes track audio volume = short fixed point level
           (mute = 0x0001 ; 100% = 1.0 ; QUICKTIME 200% max = 2.0)
       -> 2 bytes reserved = short value set to zero
       -> 4 bytes decimal video geometry matrix value A
           = long fixed point width scale (normal = 1.0)
       -> 4 bytes decimal video geometry matrix value B
           = long fixed point width rotate (normal = 0.0)
       -> 4 bytes decimal video geometry matrix value U
           = long fixed point width angle (restricted to 0.0)
       -> 4 bytes decimal video geometry matrix value C
           = long fixed point height rotate (normal = 0.0)
       -> 4 bytes decimal video geometry matrix value D
           = long fixed point height scale (normal = 1.0)
       -> 4 bytes decimal video geometry matrix value V
           = long fixed point height angle (restricted to 0.0)
       -> 4 bytes decimal video geometry matrix value X
           = long fixed point positon (left = 0.0)
       -> 4 bytes decimal video geometry matrix value Y
           = long fixed point positon (top = 0.0)
       -> 4 bytes decimal video geometry matrix value W
           = long fixed point divider scale (restricted to 1.0)
       -> 8 bytes decimal video frame size
           = long fixed point width + long fixed point height
*/
