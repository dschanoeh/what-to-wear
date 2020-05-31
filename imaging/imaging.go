package imaging

import (
	"errors"
	"fmt"
	"image"
	"os/exec"

	"github.com/MaxHalford/halfgone"
	log "github.com/sirupsen/logrus"
)

const (
	chromeScreenshotFilename = "screenshot.png"
	ditheredFilename         = "dithered.png"
)

type ImageConfig struct {
	Width        int    `yaml:"width"`
	Height       int    `yaml:"height"`
	ChromeBinary string `yaml:"chrome_binary"`
	ScrapeURL    string `yaml:"scrape_url"`
	WorkingDir   string `yaml:"working_dir"`
	Dithering    bool   `yaml:"dithering"`
}

type ImageProcessor struct {
	imageConfig  *ImageConfig
	currentImage *image.Gray
}

func New(config *ImageConfig) *ImageProcessor {
	i := ImageProcessor{imageConfig: config}
	return &i
}

func (i *ImageProcessor) takeScreenshot() error {
	_, err := exec.LookPath(i.imageConfig.ChromeBinary)
	if err != nil {
		return errors.New("didn't find Chrome executable" + i.imageConfig.ChromeBinary)
	}

	cmd := exec.Command(i.imageConfig.ChromeBinary,
		"--headless",
		"--disable-gpu",
		"--screenshot",
		fmt.Sprintf("--window-size=%d,%d", i.imageConfig.Width, i.imageConfig.Height),
		i.imageConfig.ScrapeURL,
	)

	cmd.Dir = i.imageConfig.WorkingDir
	err = cmd.Run()

	return err
}

func (i *ImageProcessor) Update() {
	err := i.takeScreenshot()
	if err != nil {
		log.Error(err)
		return
	}
	screenshot, err := halfgone.LoadImage(i.imageConfig.WorkingDir + "/" + chromeScreenshotFilename)
	if err != nil {
		log.Error(err)
		return
	}

	if i.imageConfig.Dithering {
		dithered := i.ditherImage(&screenshot)
		halfgone.SaveImagePNG(dithered, i.imageConfig.WorkingDir+"/"+ditheredFilename)
		i.currentImage = dithered
	} else {
		i.currentImage = halfgone.ImageToGray(screenshot)
	}
}

func (i *ImageProcessor) ditherImage(img *image.Image) *image.Gray {
	gray := halfgone.ImageToGray(*img)
	dithered := halfgone.TwoRowSierraDitherer{}.Apply(gray)
	return dithered
}

// GetImageAsBinary returns a one-dimensional byte array for all the pixels in the current image.
// Each bit represents one pixel.
func (i *ImageProcessor) GetImageAsBinary() []byte {
	if i.currentImage == nil {
		return nil
	}

	bounds := i.currentImage.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	data := make([]byte, width*height/8)
	index := 0

	for h := 0; h < height; h++ {
		for w := 0; w < width; w += 8 {
			// Now, we need to shrink every 8 pixels into one byte
			var b byte
			for x := 0; x < 8; x++ {
				px := i.currentImage.GrayAt(w+x, h)
				if px.Y != 0 {
					b |= 1 << (7 - x)
				}
			}
			data[index] = b
			index++
		}
	}

	return data
}
