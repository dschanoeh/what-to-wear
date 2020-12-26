package imaging

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/MaxHalford/halfgone"
	log "github.com/sirupsen/logrus"
)

const (
	chromeScreenshotFilename = "screenshot.png"
	ditheredFilename         = "dithered.png"
	chromeTimeout            = 10000 // in ms
	VirtualTimeBudget        = 5000  // in ms
	screenshotRetries        = 3
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
	tempDir      string
}

func New(config *ImageConfig) (*ImageProcessor, error) {
	tempDir, err := ioutil.TempDir("", "what-to-wear")
	if err != nil {
		return nil, err
	}
	log.Infof("Created temp dir %s", tempDir)

	i := ImageProcessor{imageConfig: config, tempDir: tempDir}

	return &i, nil
}
func (i *ImageProcessor) Close() error {
	err := os.RemoveAll(i.tempDir)
	if err != nil {
		log.Error("Couldn't delete temp dir:", err)
	}

	return err
}

func (i *ImageProcessor) takeScreenshot() error {
	_, err := exec.LookPath(i.imageConfig.ChromeBinary)
	if err != nil {
		return errors.New("didn't find Chrome executable" + i.imageConfig.ChromeBinary)
	}

	ctx, cancel := context.WithTimeout(context.Background(), chromeTimeout*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, i.imageConfig.ChromeBinary,
		"--headless",
		"--disable-gpu",
		"--disable-extensions",
		"--disable-dev-shm-usage",
		"--screenshot",
		fmt.Sprintf("--window-size=%d,%d", i.imageConfig.Width, i.imageConfig.Height),
		fmt.Sprintf("--virtual-time-budget=%d", VirtualTimeBudget),
		i.imageConfig.ScrapeURL,
	)

	cmd.Dir = i.tempDir + "/"
	log.Debug("Starting chrome...")
	out, err := cmd.Output()

	if ctx.Err() == context.DeadlineExceeded {
		log.Error("Chrome output: ", string(out))
		return errors.New("Chrome timeout")
	}

	if err != nil {
		return err
	}

	log.Debug("Chrome is done.")

	if _, err := os.Stat(i.tempDir + "/" + chromeScreenshotFilename); err == nil {
		return nil
	}

	return errors.New("Screenshot was not created")
}

func (i *ImageProcessor) Update() {
	for t := 0; t < screenshotRetries; t++ {
		log.Infof("Attempting screen capture try %d", t)
		err := i.takeScreenshot()
		if err != nil {
			log.Error(err)
			continue
		}
		screenshot, err := halfgone.LoadImage(i.tempDir + "/" + chromeScreenshotFilename)
		if err != nil {
			log.Error(err)
			continue
		}
		if isAllWhite(&screenshot) {
			log.Error("The screenshot was all white. Let's try again...")
			continue
		} else {
			if i.imageConfig.Dithering {
				dithered := i.ditherImage(&screenshot)
				halfgone.SaveImagePNG(dithered, i.imageConfig.WorkingDir+"/"+ditheredFilename)
				i.currentImage = dithered
			} else {
				i.currentImage = halfgone.ImageToGray(screenshot)
			}
			return
		}
	}
}

func (i *ImageProcessor) ditherImage(img *image.Image) *image.Gray {
	gray := halfgone.ImageToGray(*img)
	dithered := halfgone.TwoRowSierraDitherer{}.Apply(gray)
	return dithered
}

func isAllWhite(img *image.Image) bool {
	gray := halfgone.ImageToGray(*img)
	bounds := gray.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	for h := 0; h < height; h++ {
		for w := 0; w < width; w++ {
			px := gray.GrayAt(w, h)
			if px.Y != 255 {
				return false
			}
		}
	}
	return true
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
