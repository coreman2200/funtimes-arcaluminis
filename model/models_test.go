package model_test

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"periph.io/x/conn/v3/conntest"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi/spitest"
	"periph.io/x/devices/v3/nrzled"

	. "github.com/coreman2200/funtimes-arcaluminis/model"
	"github.com/stretchr/testify/assert"
)

var TestStartColorChangesToExpectedColor = []struct {
	Start  uint32
	Given  uint32
	Expect uint32
}{
	{0xFF112233, 0x00112233, 0xFF224466},
	{0x00448800, 0xFF0000FF, 0xFF4488FF},
	{0x87650000, 0x00004321, 0x87654321},
	{0x37650105, 0x30004321, 0x67654426},
}

var TestRGBIsExpectedColor = []struct {
	A      uint8
	G      uint8
	R      uint8
	B      uint8
	Expect uint32
}{
	{0xFF, 0x11, 0x22, 0x33, 0xFF112233},
	{0x00, 0x2A, 0x44, 0x34, 0x002A4434},
	{0xAB, 0x3B, 0x88, 0x35, 0xAB3B8835},
	{0x22, 0x4C, 0xAA, 0x36, 0x224CAA36},
	{0xFF, 0x5D, 0xCC, 0x37, 0xFF5DCC37},
}

func EntryBitRepresentation(c ColorVal) {
	fmt.Println("Color:" + strconv.FormatInt(int64(c.Color()), 2) + "(0x" + strconv.FormatInt(int64(c.Color()), 16) + ")")
	fmt.Println("Red:" + strconv.FormatInt(int64(c.GetR()), 2) + "(0x" + strconv.FormatInt(int64(c.GetR()), 16) + ")")
	fmt.Println("Green:" + strconv.FormatInt(int64(c.GetG()), 2) + "(0x" + strconv.FormatInt(int64(c.GetG()), 16) + ")")
	fmt.Println("Blue:" + strconv.FormatInt(int64(c.GetB()), 2) + "(0x" + strconv.FormatInt(int64(c.GetB()), 16) + ")")
	fmt.Println("Alpha:" + strconv.FormatInt(int64(c.GetA()), 2) + "(0x" + strconv.FormatInt(int64(c.GetA()), 16) + ")")
}

func TestSPI_Empty(t *testing.T) {
	buf := bytes.Buffer{}
	o := nrzled.Opts{NumPixels: 0, Channels: 3, Freq: 2500 * physic.KiloHertz}
	s := spitest.Playback{
		Playback: conntest.Playback{
			Count: 1,
			Ops:   []conntest.IO{{W: []byte{0x00, 0x00, 0x00}}},
		},
	}
	d, err := nrzled.NewSPI(spitest.NewRecordRaw(&buf), &o)
	if err != nil {
		t.Fatal(err)
	}
	if got, expected := d.String(), "nrzled{recordraw}"; got != expected {
		t.Fatalf("\nGot:  %s\nWant: %s\n", got, expected)
	}

	if n, err := d.Write([]byte{}); n != 0 || err != nil {
		t.Fatalf("%d %v", n, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestColorsRGB(t *testing.T) {
	for k, v := range TestRGBIsExpectedColor {
		t.Run("Given RGB"+strconv.FormatUint(uint64(k), 10), func(t *testing.T) {
			col := NewColor(0)
			col.SetA(v.A)
			col.SetR(v.R)
			col.SetG(v.G)
			col.SetB(v.B)
			EntryBitRepresentation(col)
			assert.Equal(t, col.Color(), v.Expect, "should be same val")
		},
		)
	}
}

func TestColorsChanges(t *testing.T) {
	for k, v := range TestStartColorChangesToExpectedColor {
		t.Run("Given RGB"+strconv.FormatUint(uint64(k), 10), func(t *testing.T) {
			col1 := NewColor(v.Start)
			col2 := NewColor(v.Given)

			EntryBitRepresentation(col1)

			col1.SetA(col1.GetA() + col2.GetA())
			col1.SetR(col1.GetR() + col2.GetR())
			col1.SetG(col1.GetG() + col2.GetG())
			col1.SetB(col1.GetB() + col2.GetB())

			EntryBitRepresentation(col2)

			assert.Equal(t, col1.Color(), v.Expect, "should be same val")
		},
		)
	}
}
