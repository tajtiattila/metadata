package exif

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestTagGetters(t *testing.T) {

	testTagGetter(t, TypeByte, 2, []byte{1, 2}, []byte{1, 2}, func(tag *Tag) {
		have := tag.Byte()
		want := []byte{1, 2}
		if !bytes.Equal(have, want) {
			t.Errorf("TestTagGetters %s got %v, want %v", typeStr(TypeByte), have, want)
		}
	})

	testTagGetter(t, TypeUndef, 2, []byte{1, 2}, []byte{1, 2}, func(tag *Tag) {
		have := tag.Undef()
		want := []byte{1, 2}
		if !bytes.Equal(have, want) {
			t.Errorf("TestTagGetters %s got %v, want %v", typeStr(TypeUndef), have, want)
		}
	})

	testTagGetter(t, TypeAscii, 6, []byte("hello\x00"), []byte("hello\x00"), func(tag *Tag) {
		have, ok := tag.Ascii()
		want := "hello"
		if !ok || have != want {
			t.Errorf("TestTagGetters %s got %v, want %v", typeStr(TypeAscii), have, want)
		}
	})

	testTagGetter(t, TypeShort, 1, []byte{1, 2}, []byte{2, 1}, func(tag *Tag) {
		have := tag.Short()
		want := []uint16{0x102}
		if len(have) != len(want) {
			t.Errorf("TestTagGetters %s got %v, want %v", typeStr(TypeShort), have, want)
			return
		}
		for i := range have {
			if have[i] != want[i] {
				t.Errorf("TestTagGetters %s got %v, want %v", typeStr(TypeShort), have, want)
				return
			}
		}
	})

	testTagGetter(t, TypeLong, 1, []byte{1, 2, 3, 4}, []byte{4, 3, 2, 1}, func(tag *Tag) {
		have := tag.Long()
		want := []uint32{0x1020304}
		if len(have) != len(want) {
			t.Errorf("TestTagGetters %s got %v, want %v", typeStr(TypeLong), have, want)
			return
		}
		for i := range have {
			if have[i] != want[i] {
				t.Errorf("TestTagGetters %s got %v, want %v", typeStr(TypeLong), have, want)
				return
			}
		}
	})

	testTagGetter(t, TypeSLong, 1, []byte{1, 2, 3, 4}, []byte{4, 3, 2, 1}, func(tag *Tag) {
		have := tag.SLong()
		want := []int32{0x1020304}
		if len(have) != len(want) {
			t.Errorf("TestTagGetters %s got %v, want %v", typeStr(TypeSLong), have, want)
			return
		}
		for i := range have {
			if have[i] != want[i] {
				t.Errorf("TestTagGetters %s got %v, want %v", typeStr(TypeSLong), have, want)
				return
			}
		}
	})

	testTagGetter(t, TypeRational, 1,
		[]byte{1, 2, 3, 4, 5, 6, 7, 8},
		[]byte{4, 3, 2, 1, 8, 7, 6, 5},
		func(tag *Tag) {
			have := tag.Rational()
			want := []uint32{0x1020304, 0x5060708}
			if len(have) != len(want) {
				t.Errorf("TestTagGetters %s got %v, want %v", typeStr(TypeRational), have, want)
				return
			}
			for i := range have {
				if have[i] != want[i] {
					t.Errorf("TestTagGetters %s got %v, want %v", typeStr(TypeRational), have, want)
					return
				}
			}
		})
}

func testTagGetter(t *testing.T, typ uint16, count uint32, beraw, leraw []byte, f func(tag *Tag)) {
	testTagGetterBo(t, binary.BigEndian, typ, count, beraw, f)
	testTagGetterBo(t, binary.LittleEndian, typ, count, leraw, f)
}

func testTagGetterBo(t *testing.T, bo binary.ByteOrder, typ uint16, count uint32, raw []byte, f func(tag *Tag)) {
	tag := &Tag{
		ByteOrder: bo,
		E: Entry{
			Type:  typ,
			Count: count,
			Value: raw,
		},
	}
	if !tag.Valid() {
		t.Errorf("tag %s thinks it is invalid", typeStr(typ))
		return
	}
	if !tag.IsType(typ) {
		t.Errorf("tag thinks it is not %s", typeStr(typ))
		return
	}
	f(tag)
}

func typeStr(typ uint16) string {
	switch typ {
	case TypeByte:
		return "Byte"
	case TypeAscii:
		return "Ascii"
	case TypeShort:
		return "Short"
	case TypeLong:
		return "Long"
	case TypeRational:
		return "Rational"
	case TypeUndef:
		return "Undef"
	case TypeSLong:
		return "SLong"
	case TypeSRational:
		return "SRational"
	}
	return "«invalid»"
}
