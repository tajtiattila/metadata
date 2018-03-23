package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"text/template"
	"time"

	"github.com/tajtiattila/metadata"
)

func main() {
	gpx := flag.Bool("gpx", false, "gpx output")
	flag.Parse()

	w := walker{}
	for _, a := range flag.Args() {
		w.walk(a)
	}

	sort.Sort(bytime(w.pt))

	pts := dedup(w.pt)

	if *gpx {
		t := template.Must(template.New("gpx").Parse(gpxt))
		t.Execute(os.Stdout, pts)
	} else {
		for _, p := range pts {
			fmt.Printf("%s %11.6f %11.6f\n", p.t.Format(time.RFC3339), p.Lat, p.Lon)
		}
	}
}

type walker struct {
	pt []point
}

func (w *walker) walk(root string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("Walk error:", err)
			return nil
		}
		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			log.Println(err)
			return nil
		}
		defer f.Close()
		meta, err := metadata.Parse(f)
		if err != nil {
			if err != metadata.ErrUnknownFormat {
				log.Println(err)
			}
			return nil
		}

		if err := w.record(meta); err != nil {
			log.Println(path, err)
		}

		return nil
	})
}

func (w *walker) record(meta *metadata.Metadata) error {
	a := meta.Attr
	slat := a[metadata.GPSLatitude]
	slon := a[metadata.GPSLongitude]
	st := a[metadata.GPSDateTime]
	if slat == "" && slon == "" {
		return nil
	}
	lat, err := strconv.ParseFloat(slat, 64)
	if err != nil {
		return fmt.Errorf("Can't parse latitude %q", slat)
	}
	lon, err := strconv.ParseFloat(slon, 64)
	if err != nil {
		return fmt.Errorf("Can't parse longitude %q", slon)
	}
	if st == "" {
		return fmt.Errorf("GPS time missing")
	}
	t, err := time.Parse(time.RFC3339, st)
	if err != nil {
		return fmt.Errorf("Can't parse GPS time %q: %v", st, err)
	}
	w.pt = append(w.pt, pt(t, lat, lon))
	return nil
}

type point struct {
	t        time.Time
	Lat, Lon float64
}

func pt(t time.Time, lat, lon float64) point {
	return point{t, lat, lon}
}

func (p point) TimeZ() string {
	return p.t.Format(time.RFC3339)
}

type bytime []point

func (b bytime) Len() int           { return len(b) }
func (b bytime) Less(i, j int) bool { return b[i].t.Before(b[j].t) }
func (b bytime) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func dedup(v []point) []point {
	j := 1
	for _, p := range v[1:] {
		if p != v[j-1] {
			v[j], j = p, j+1
		}
	}
	return v[:j]
}

var gpxt = `<?xml version="1.0" encoding="utf-8"?>
<gpx version="1.0"
 xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
 xmlns="http://www.topografix.com/GPX/1/0"
 xsi:schemaLocation="http://www.topografix.com/GPX/1/0 http://www.topografix.com/GPX/1/0/gpx.xsd">
<trk>
<trkseg>
{{range . -}}
<trkpt lat="{{.Lat}}" lon="{{.Lon}}"><time>{{.TimeZ}}</time></trkpt>
{{end -}}
</trkseg>
</trk>
</gpx>
`
