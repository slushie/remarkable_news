package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"strconv"
	"strings"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

const (
	dpi             = 226
	defaultFontSpec = "/usr/share/fonts/ttf/noto/NotoSans-Regular.ttf:12"
	overlayPadding  = 10
)

var fontCache = map[string]font.Face{}

type drawColor string

const (
	transparent drawColor = `transparent`
	black       drawColor = `black`
	gray1       drawColor = `gray1`
	gray2       drawColor = `gray2`
	white       drawColor = `white`
)

type textOverlay struct {
	// drawing points relative to image size (ie, 0,0 is top left)
	x, y int

	// text and background colors
	fg, bg drawColor

	// text font
	font font.Face

	// string s
	s string
}

type textOverlayList []textOverlay

func (t *textOverlayList) Set(val string) error {
	to := textOverlay{
		fg: black,
		bg: white,
		s:  "<no content>",
	}
	pairs := strings.Split(val, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("cannot parse %q", pair)
		}
		var err error
		switch kv[0] {
		case "x":
			x, err := strconv.ParseInt(strings.TrimSuffix(kv[1], "%"), 10, 0)
			if err != nil {
				return fmt.Errorf("parse %q failed: %+v", pair, err)
			}
			if strings.HasSuffix(kv[1], "%") {
				x = x / 100.0 * re_width
			}
			to.x = int(x)
		case "y":
			y, err := strconv.ParseInt(strings.TrimSuffix(kv[1], "%"), 10, 0)
			if err != nil {
				return fmt.Errorf("parse %q failed: %+v", pair, err)
			}
			if strings.HasSuffix(kv[1], "%") {
				y = y / 100.0 * re_height
			}
			to.y = int(y)
		case "fg":
			to.fg, err = parseColor(kv[1])
			if to.fg == transparent {
				return fmt.Errorf("invalid fg color: %v", to.fg)
			}
			if err != nil {
				return fmt.Errorf("parse %q failed: %+v", pair, err)
			}
		case "bg":
			to.bg, err = parseColor(kv[1])
			if err != nil {
				return fmt.Errorf("parse %q failed: %+v", pair, err)
			}
		case "font":
			to.font, err = parseFont(kv[1])
			if err != nil {
				return fmt.Errorf("parse %q failed: %+v", pair, err)
			}
		case "string", "str", "s":
			to.s = kv[1]
		default:
			return fmt.Errorf("parse failed: unknown key %v", kv[0])
		}
	}

	// default font
	if to.font == nil {
		to.font, _ = parseFont(defaultFontSpec)
	}

	*t = append(*t, to)
	return nil
}

func (t textOverlayList) String() string {
	return fmt.Sprint([]textOverlay(t))
}

func parseColor(v string) (drawColor, error) {
	c := drawColor(v)
	switch c {
	case black, white, gray1, gray2, transparent:
		return c, nil
	case "":
		return transparent, nil
	default:
		return "", fmt.Errorf("invalid color %q", v)
	}
}

func parseFont(v string) (font.Face, error) {
	if fp, ok := fontCache[v]; ok {
		return fp, nil
	}

	f := strings.SplitN(v, ":", 2)
	if len(f) != 2 {
		debug("Default font size 12")
		f = append(f, "12")
	}

	size, err := strconv.ParseFloat(f[1], 64)
	if err != nil {
		return nil, err
	}

	b, err := os.ReadFile(f[0])
	if err != nil {
		return nil, err
	}

	ttf, err := freetype.ParseFont(b)
	if err != nil {
		return nil, err
	}

	face := truetype.NewFace(ttf, &truetype.Options{
		Size: size,
		DPI:  226.0,
	})
	fontCache[v] = face

	return face, nil
}

func (c drawColor) Uniform() image.Image {
	var img image.Image
	switch c {
	case black:
		img = image.Black
	case white:
		img = image.White
	case gray1:
		img = image.NewUniform(color.Gray{85})
	case gray2:
		img = image.NewUniform(color.Gray{170})
	}
	return img
}

func overlay(img image.Image, to textOverlay) (image.Image, error) {
	dst, ok := img.(draw.Image)
	if !ok {
		return nil, fmt.Errorf("image is immutable")
	}

	if to.bg != transparent {
		bg := to.bg.Uniform()
		extent, _ := font.BoundString(to.font, to.s)
		pt := image.Pt(int(to.x), int(to.y))
		r := image.Rect(
			pt.X+(extent.Min.X.Round())-overlayPadding,
			pt.Y+(extent.Min.Y.Round())-overlayPadding,
			pt.X+(extent.Max.X.Round())+overlayPadding,
			pt.Y+(extent.Max.Y.Round())+overlayPadding,
		)
		draw.Draw(dst, r, bg, pt, draw.Src)
	}

	fg := to.fg.Uniform()
	d := font.Drawer{
		Dst:  dst,
		Src:  fg,
		Face: to.font,
		Dot:  freetype.Pt(int(to.x), int(to.y)),
	}
	d.DrawString(to.s)
	return img, nil
}
