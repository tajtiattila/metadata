package xmldoc_test

import (
	"bytes"
	"encoding/xml"
	"os"
	"testing"

	"github.com/tajtiattila/metadata/xmldoc"
)

func TestEncodeDecode(t *testing.T) {
	decoder := xml.NewDecoder(bytes.NewReader([]byte(sample)))
	var root xmldoc.Node
	if err := decoder.Decode(&root); err != nil {
		t.Fatal("can't decode:", err)
	}

	dump(t, &root)

	root.Translate()

	t.Log("translated:")
	dump(t, &root)

	enc := xml.NewEncoder(os.Stdout)
	enc.Indent("", "  ")
	err := enc.Encode(&root)
	if err != nil {
		t.Fatal("can't encode", err)
	}
}

func dump(t *testing.T, n *xmldoc.Node) {
	for {
		t.Logf("%#v", n)
		if len(n.Child) != 0 {
			n = n.Child[0]
		} else {
			break
		}
	}
}

const sample = `<?xpacket begin='` + "\ufeff" + `' id='W5M0MpCehiHzreSzNTczkc9d'?>
<x:xmpmeta xmlns:x='adobe:ns:meta/' x:xmptk='Image::ExifTool 10.17'>
<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>

 <rdf:Description rdf:about=''
  xmlns:exif='http://ns.adobe.com/exif/1.0/'>
  <exif:ApertureValue>4845/1918</exif:ApertureValue>
  <exif:ColorSpace>1</exif:ColorSpace>
  <exif:ComponentsConfiguration>
   <rdf:Seq>
    <rdf:li>1</rdf:li>
    <rdf:li>2</rdf:li>
    <rdf:li>3</rdf:li>
    <rdf:li>0</rdf:li>
   </rdf:Seq>
  </exif:ComponentsConfiguration>
  <exif:DateTimeOriginal>2014-07-11T08:44:34</exif:DateTimeOriginal>
  <exif:ExifVersion>0220</exif:ExifVersion>
  <exif:ExposureBiasValue>0/1</exif:ExposureBiasValue>
  <exif:ExposureTime>1/148</exif:ExposureTime>
  <exif:FNumber>12/5</exif:FNumber>
  <exif:Flash rdf:parseType='Resource'>
   <exif:Fired>False</exif:Fired>
   <exif:Function>False</exif:Function>
   <exif:Mode>0</exif:Mode>
   <exif:RedEyeMode>False</exif:RedEyeMode>
   <exif:Return>0</exif:Return>
  </exif:Flash>
  <exif:FlashpixVersion>0100</exif:FlashpixVersion>
  <exif:FocalLength>4/1</exif:FocalLength>
  <exif:GPSAltitude>0/1</exif:GPSAltitude>
  <exif:GPSAltitudeRef>0</exif:GPSAltitudeRef>
  <exif:GPSImgDirection>181/1</exif:GPSImgDirection>
  <exif:GPSImgDirectionRef>M</exif:GPSImgDirectionRef>
  <exif:GPSLatitude>37,45.089950N</exif:GPSLatitude>
  <exif:GPSLongitude>122,25.767517W</exif:GPSLongitude>
  <exif:GPSProcessingMethod>ASCII</exif:GPSProcessingMethod>
  <exif:GPSTimeStamp>2014-07-11T15:44:32Z</exif:GPSTimeStamp>
  <exif:ISOSpeedRatings>
   <rdf:Seq>
    <rdf:li>100</rdf:li>
   </rdf:Seq>
  </exif:ISOSpeedRatings>
  <exif:PixelXDimension>204</exif:PixelXDimension>
  <exif:PixelYDimension>153</exif:PixelYDimension>
  <exif:ShutterSpeedValue>39962/5543</exif:ShutterSpeedValue>
 </rdf:Description>

 <rdf:Description rdf:about=''
  xmlns:exifEX='http://cipa.jp/exif/1.0/'>
  <exifEX:InteroperabilityIndex>R98</exifEX:InteroperabilityIndex>
 </rdf:Description>

 <rdf:Description rdf:about=''
  xmlns:tiff='http://ns.adobe.com/tiff/1.0/'>
  <tiff:BitsPerSample>
   <rdf:Seq>
    <rdf:li>8</rdf:li>
   </rdf:Seq>
  </tiff:BitsPerSample>
  <tiff:ImageLength>153</tiff:ImageLength>
  <tiff:ImageWidth>204</tiff:ImageWidth>
  <tiff:Make>LGE</tiff:Make>
  <tiff:Model>Nexus 5</tiff:Model>
  <tiff:ResolutionUnit>2</tiff:ResolutionUnit>
  <tiff:XResolution>72/1</tiff:XResolution>
  <tiff:YCbCrPositioning>1</tiff:YCbCrPositioning>
  <tiff:YCbCrSubSampling>
   <rdf:Seq>
    <rdf:li>1</rdf:li>
    <rdf:li>1</rdf:li>
   </rdf:Seq>
  </tiff:YCbCrSubSampling>
  <tiff:YResolution>72/1</tiff:YResolution>
 </rdf:Description>

 <rdf:Description rdf:about=''
  xmlns:xmp='http://ns.adobe.com/xap/1.0/'>
  <xmp:CreateDate>2014-07-11T08:44:34</xmp:CreateDate>
 </rdf:Description>
</rdf:RDF>
</x:xmpmeta>
<?xpacket end='w'?>`
