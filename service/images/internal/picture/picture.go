package picture

import (
	"bufio"
	"image"
	"image/jpeg"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"github.com/nfnt/resize"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
)

// Resize picture. Attention! After using this is function need move to start of the 'in' reader
func Resize(out io.Writer, in io.Reader, width, height uint) error {

	sr := bufio.NewReader(in)
	img, _, err := image.Decode(sr)
	if err != nil {
		return err
	}

	newImg := resize.Resize(width, height, img, resize.Lanczos3)

	return jpeg.Encode(out, newImg, &jpeg.Options{Quality: 100})
}
