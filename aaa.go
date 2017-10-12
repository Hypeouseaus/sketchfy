/*
NAME
  sketch - sketch an image or video

SYNOPSIS
  sketch [-framelimit -iter -l -p -save -start -stat] [file]
  ffmpeg -i input.webm input_%03d.png && sketch && ffmpeg -i frame_%03d.png output.webm

DESCRIPTION
  Sketch approximates input images using randomly placed lines.

  The -p flag removes duplicate colours from the palette, which means a more
  uniformly random selection of colours is used to draw lines. Some images,
  like line art, may converge faster with the -p flag enabled.

  -framelimit limit
        limit for total number of output frames
  -iter limit
        iteration limit (-1 for infinite) (default 5000000)
  -l length
        line length limit (default 40)
  -p    remove duplicate colours from palette
  -save interval
        incremental save interval, in seconds (default -1)
  -start int
        starting frame number (default 1)
  -stat interval
        statistics reporting interval, in seconds (default 1)
*/
package main

import (
	"flag"
	"fmt"
	"github.com/StephaneBunel/bresenham"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"time"
)

func bdiff(a, b image.Image, x1, y1, x2, y2 int) float64 {
	var dx, dy, e, slope int
	var dif float64

	if x1 > x2 {
		x1, y1, x2, y2 = x2, y2, x1, y1
	}

	dx, dy = x2-x1, y2-y1
	if dy < 0 {
		dy = -dy
	}

	switch {
	case x1 == x2 && y1 == y2:
		dif += calcdiff(a, b, x1, y1)
	case y1 == y2:
		for ; dx != 0; dx-- {
			dif += calcdiff(a, b, x1, y1)
			x1++
		}
		dif += calcdiff(a, b, x1, y1)
	case x1 == x2:
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		for ; dy != 0; dy-- {
			dif += calcdiff(a, b, x1, y1)
			y1++
		}
		dif += calcdiff(a, b, x1, y1)
	case dx == dy:
		if y1 < y2 {
			for ; dx != 0; dx-- {
				dif += calcdiff(a, b, x1, y1)
				x1++
				y1++
			}
		} else {
			for ; dx != 0; dx-- {
				dif += calcdiff(a, b, x1, y1)
				x1++
				y1--
			}
		}
		dif += calcdiff(a, b, x1, y1)
	case dx > dy:
		if y1 < y2 {
			dy, e, slope = 2*dy, dx, 2*dx
			for ; dx != 0; dx-- {
				dif += calcdiff(a, b, x1, y1)
				x1++
				e -= dy
				if e < 0 {
					y1++
					e += slope
				}
			}
		} else {
			dy, e, slope = 2*dy, dx, 2*dx
			for ; dx != 0; dx-- {
				dif += calcdiff(a, b, x1, y1)
				x1++
				e -= dy
				if e < 0 {
					y1--
					e += slope
				}
			}
		}
		dif += calcdiff(a, b, x2, y2)
	default:
		if y1 < y2 {
			dx, e, slope = 2*dx, dy, 2*dy
			for ; dy != 0; dy-- {
				dif += calcdiff(a, b, x1, y1)
				y1++
				e -= dx
				if e < 0 {
					x1++
					e += slope
				}
			}
		} else {
			dx, e, slope = 2*dx, dy, 2*dy
			for ; dy != 0; dy-- {
				dif += calcdiff(a, b, x1, y1)
				y1--
				e -= dx
				if e < 0 {
					x1++
					e += slope
				}
			}
		}
		dif += calcdiff(a, b, x2, y2)
	}
	return dif
}

func calcdiff(a, b image.Image, x, y int) float64 {
	aR, aG, aB, aA := a.At(x, y).RGBA()
	bR, bG, bB, bA := b.At(x, y).RGBA()
	ra := float64(aR)
	rb := float64(bR)
	ga := float64(aG)
	gb := float64(bG)
	ba := float64(aB)
	bb := float64(bB)
	aa := float64(aA)
	ab := float64(bA)
	R := (rb - ra) * (rb - ra)
	G := (gb - ga) * (gb - ga)
	B := (bb - ba) * (bb - ba)
	A := (ab - aa) * (ab - aa)
	return math.Sqrt(R + G + B + A)
}

func bcopy(img, src *image.RGBA, x1, y1, x2, y2 int) {
	var dx, dy, e, slope int

	if x1 > x2 {
		x1, y1, x2, y2 = x2, y2, x1, y1
	}

	dx, dy = x2-x1, y2-y1
	if dy < 0 {
		dy = -dy
	}

	switch {
	case x1 == x2 && y1 == y2:
		img.Set(x1, y1, src.At(x1, y1))
	case y1 == y2:
		for ; dx != 0; dx-- {
			img.Set(x1, y1, src.At(x1, y1))
			x1++
		}
		img.Set(x1, y1, src.At(x1, y1))
	case x1 == x2:
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		for ; dy != 0; dy-- {
			img.Set(x1, y1, src.At(x1, y1))
			y1++
		}
		img.Set(x1, y1, src.At(x1, y1))
	case dx == dy:
		if y1 < y2 {
			for ; dx != 0; dx-- {
				img.Set(x1, y1, src.At(x1, y1))
				x1++
				y1++
			}
		} else {
			for ; dx != 0; dx-- {
				img.Set(x1, y1, src.At(x1, y1))
				x1++
				y1--
			}
		}
		img.Set(x1, y1, src.At(x1, y1))
	case dx > dy:
		if y1 < y2 {
			dy, e, slope = 2*dy, dx, 2*dx
			for ; dx != 0; dx-- {
				img.Set(x1, y1, src.At(x1, y1))
				x1++
				e -= dy
				if e < 0 {
					y1++
					e += slope
				}
			}
		} else {
			dy, e, slope = 2*dy, dx, 2*dx
			for ; dx != 0; dx-- {
				img.Set(x1, y1, src.At(x1, y1))
				x1++
				e -= dy
				if e < 0 {
					y1--
					e += slope
				}
			}
		}
		img.Set(x2, y2, src.At(x2, y2))
	default:
		if y1 < y2 {
			dx, e, slope = 2*dx, dy, 2*dy
			for ; dy != 0; dy-- {
				img.Set(x1, y1, src.At(x1, y1))
				y1++
				e -= dx
				if e < 0 {
					x1++
					e += slope
				}
			}
		} else {
			dx, e, slope = 2*dx, dy, 2*dy
			for ; dy != 0; dy-- {
				img.Set(x1, y1, src.At(x1, y1))
				y1--
				e -= dx
				if e < 0 {
					x1++
					e += slope
				}
			}
		}
		img.Set(x2, y2, src.At(x2, y2))
	}
}

func save(img image.Image, name string) {
	name = fmt.Sprintf("%s.png", name)
	outf, err := os.Create(name)
	if err != nil {
		log.Fatalln(err)
	}
	defer outf.Close()
	png.Encode(outf, img)
	log.Println("wrote", name)
}

var iterLimit int
var frameStart int
var frameLimit int
var lineLen int
var palletize bool
var saveInterval float64
var statInterval float64

func init() {
	flag.IntVar(&iterLimit, "iter", 5000000, "iteration `limit` (-1 for infinite)")
	flag.IntVar(&frameStart, "start", 1, "starting frame number")
	flag.IntVar(&frameLimit, "framelimit", 0, "`limit` for total number of output frames")
	flag.IntVar(&lineLen, "l", 40, "line `length` limit")
	flag.BoolVar(&palletize, "p", false, "remove duplicate colours from palette")
	flag.Float64Var(&saveInterval, "save", -1.0, "save `interval`, in seconds")
	flag.Float64Var(&statInterval, "stat", 1.0, "statistics reporting `interval`, in seconds")
}

var incrSaveNum = 1 // when saving incrementally
var saveNum = 1     // when saving finished frames

func sketch(src image.Image) {
	w := src.Bounds().Dx()
	h := src.Bounds().Dy()

	img := image.NewRGBA(src.Bounds())
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			clr := src.At(x, y)
			clr = color.RGBAModel.Convert(clr)
			img.Set(x, y, clr)
		}
	}

	palette := make([]color.Color, 0, 600000)
	palettemap := make(map[color.Color]bool, 600000)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if palletize {
				if _, ok := palettemap[img.At(x, y)]; !ok {
					palette = append(palette, img.At(x, y))
					palettemap[img.At(x, y)] = true
				}
			} else {
				palette = append(palette, img.At(x, y))
			}
		}
	}
	log.Printf("%d colours in palette\n", len(palette))

	img1 := image.NewRGBA(img.Bounds())
	img2 := image.NewRGBA(img.Bounds())
	bg := color.RGBA{0, 0, 0, 255}
	draw.Draw(img1, img1.Bounds(), &image.Uniform{bg}, image.ZP, draw.Src)
	draw.Draw(img2, img2.Bounds(), &image.Uniform{bg}, image.ZP, draw.Src)

	var lastSaveTime = time.Now()
	var lastStatTime = time.Now()
	var stati int
	var statc int

	for i := 0; i < iterLimit || iterLimit < 0; i++ {
		stati++
		x1 := rand.Intn(w)
		y1 := rand.Intn(h)
		x2 := -lineLen/2 + x1 + rand.Intn(lineLen)
		y2 := -lineLen/2 + y1 + rand.Intn(lineLen)
		//x2 := x1 + lineLen + rand.Intn(10)
		//y2 := y1 + lineLen/2 + rand.Intn(10)
		clr := palette[rand.Intn(len(palette))]

		bresenham.Bresenham(img1, x1, y1, x2, y2, clr)

		if bdiff(img, img1, x1, y1, x2, y2) < bdiff(img, img2, x1, y1, x2, y2) {
			// converges
			bcopy(img2, img1, x1, y1, x2, y2)
			statc++
		} else {
			// diverges
			bcopy(img1, img2, x1, y1, x2, y2)
		}
		if i%50 == 0 { // don't smash that time.Now()
			now := time.Now()
			dur := now.Sub(lastSaveTime)
			if saveInterval > 0 && dur >= time.Duration(saveInterval)*time.Second {
				save(img2, fmt.Sprintf("incr_%03d", incrSaveNum))
				incrSaveNum++
				lastSaveTime = now
			}
			dur = now.Sub(lastStatTime)
			if dur >= time.Duration(statInterval)*time.Second {
				ips := float64(stati) / dur.Seconds()
				cps := float64(statc) / dur.Seconds()
				log.Printf("%8d iters %10.2f iter/s %9.2f converg/s %6.2f%% c/i\n", i, ips, cps, 100*cps/ips)
				stati = 0
				statc = 0
				lastStatTime = now
			}
		}
	}

	save(img2, fmt.Sprintf("frame_%03d", saveNum))
	saveNum++
}

func main() {
	log.SetFlags(0)
	rand.Seed(1234)
	flag.Parse()
	//if flag.NArg() != 1 {
	//	log.Fatalln("usage: sketch [-iter -l -p -save -stat] [file]")
	//}

	frameNum := frameStart

	for {
		if frameLimit > 1 && frameNum-frameStart > frameLimit {
			break
		}
		log.Println(fmt.Sprintf("looking for input_%03d.png", frameNum))
		f, err := os.Open(fmt.Sprintf("input_%03d.png", frameNum))
		frameNum++
		if err != nil {
			break
			//log.Fatalln(err)
		}
		src, _, err := image.Decode(f)
		if err != nil {
			log.Fatalln(err)
		}
		f.Close()

		sketch(src)
	}
	log.Println("end of frames")
}