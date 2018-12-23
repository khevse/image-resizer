package picture

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResize(t *testing.T) {

	{
		res := bytes.NewBuffer(nil)
		require.NoError(t, Resize(res, helperNewImage(t, 640, 480), 640, 480))
		require.Equal(t, "fc3896ff036b8ce1b22726095decd9c2", helperMD5(t, res))
	}

	{
		res := bytes.NewBuffer(nil)
		require.NoError(t, Resize(res, helperNewImage(t, 640, 480), 200, 200))
		require.Equal(t, "eaee52177384ef106128b029adddf64e", helperMD5(t, res))
	}

	{
		res := bytes.NewBuffer(nil)
		require.NoError(t, Resize(res, helperNewImage(t, 480, 700), 200, 200))
		require.Equal(t, "13a68fb451bea8f7c6556045e2499355", helperMD5(t, res))
	}

	{
		res := bytes.NewBuffer(nil)
		require.NoError(t, Resize(res, helperNewImage(t, 640, 480), 10, 10))
		require.Equal(t, "f8270841afe8b537fc0699a36e6ca7d1", helperMD5(t, res))
	}
}

func helperNewImage(t *testing.T, width, height int) *bytes.Buffer {
	t.Helper()

	// Generate source image

	tmpImage := image.NewRGBA(image.Rect(0, 0, width, height))
	blue := color.RGBA{0, 0, 255, 255}
	draw.Draw(tmpImage, tmpImage.Bounds(), &image.Uniform{blue}, image.ZP, draw.Src)

	// Write white line
	white := color.RGBA{255, 255, 255, 255}
	middle := tmpImage.Bounds().Max.Y / 2
	for i := tmpImage.Bounds().Min.X; i < tmpImage.Bounds().Max.X; i++ {
		for j := middle - int(middle/3); j < middle+int(middle/3); j++ {
			tmpImage.Set(i, j, white)
		}
	}

	out := bytes.NewBuffer(nil)
	opt := jpeg.Options{Quality: 100}
	require.NoError(t, jpeg.Encode(out, tmpImage, &opt))

	return out
}

func helperMD5(t *testing.T, src *bytes.Buffer) string {
	t.Helper()

	hash := md5.Sum(src.Bytes())

	buf := make([]byte, len(hash)*2)
	hex.Encode(buf, hash[:])

	return string(buf)
}
