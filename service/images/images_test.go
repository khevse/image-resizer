package images

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResize(t *testing.T) {

	mux := http.NewServeMux()
	mux.Handle("/", New().Mux())
	mux.HandleFunc("/source", func(w http.ResponseWriter, req *http.Request) {

		buf := helperNewImage(t, 1000, 1000)
		_, err := io.Copy(w, buf)
		require.NoError(t, err)
	})

	testSvr := httptest.NewServer(mux)
	defer testSvr.Close()

	u, err := url.Parse(testSvr.URL + "/resize")
	require.NoError(t, err)

	for _, testinfo := range []struct {
		Name string
		Fn   func(t *testing.T)
	}{
		{"Ok", func(*testing.T) { testResizeOk(t, testSvr) }},
		{"invalid URL", func(*testing.T) { testResizeInvalidUrl(t, u) }},
		{"invalid with", func(*testing.T) { testResizeInvalidWith(t, u) }},
		{"invalid height", func(*testing.T) { testResizeInvalidHeight(t, u) }},
	} {
		if !t.Run(testinfo.Name, testinfo.Fn) {
			return
		}
	}
}

func testResizeInvalidHeight(t *testing.T, u *url.URL) {

	{
		helperSetQuery(u, "url", "-", "width", "1", "height", "")
		res, err := http.Get(u.String())
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		require.Equal(t,
			`invalid property height: strconv.Atoi: parsing "": invalid syntax`+"\n",
			helperGetStringFromBody(t, res))
	}

	{
		helperSetQuery(u, "url", "-", "width", "1", "height", "a")
		res, err := http.Get(u.String())
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		require.Equal(t,
			`invalid property height: strconv.Atoi: parsing "a": invalid syntax`+"\n",
			helperGetStringFromBody(t, res))
	}

	{
		helperSetQuery(u, "url", "-", "width", "1", "height", "1.2")
		res, err := http.Get(u.String())
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		require.Equal(t,
			`invalid property height: strconv.Atoi: parsing "1.2": invalid syntax`+"\n",
			helperGetStringFromBody(t, res))
	}
}

func testResizeInvalidWith(t *testing.T, u *url.URL) {

	{
		helperSetQuery(u, "url", "-", "width", "", "height", "")
		res, err := http.Get(u.String())
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		require.Equal(t,
			`invalid property width: strconv.Atoi: parsing "": invalid syntax`+"\n",
			helperGetStringFromBody(t, res))
	}

	{
		helperSetQuery(u, "url", "-", "width", "a", "height", "")
		res, err := http.Get(u.String())
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		require.Equal(t,
			`invalid property width: strconv.Atoi: parsing "a": invalid syntax`+"\n",
			helperGetStringFromBody(t, res))
	}

	{
		helperSetQuery(u, "url", "-", "width", "1.2", "height", "")
		res, err := http.Get(u.String())
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		require.Equal(t,
			`invalid property width: strconv.Atoi: parsing "1.2": invalid syntax`+"\n",
			helperGetStringFromBody(t, res))
	}
}

func testResizeInvalidUrl(t *testing.T, u *url.URL) {

	{
		helperSetQuery(u, "url", "", "width", "20", "height", "20")
		res, err := http.Get(u.String())
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		require.Equal(t,
			`invalid resource URL`+"\n",
			helperGetStringFromBody(t, res))
	}

	{
		helperSetQuery(u, "url", "-", "width", "20", "height", "20")
		res, err := http.Get(u.String())
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		require.Equal(t,
			`failed to send request:Get -: unsupported protocol scheme ""`+"\n",
			helperGetStringFromBody(t, res))
	}

	{
		helperSetQuery(u, "url", "https://google.com", "width", "20", "height", "20")
		res, err := http.Get(u.String())
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, res.StatusCode)
		require.Equal(t,
			`internal server error:image: unknown format`+"\n",
			helperGetStringFromBody(t, res))
	}
}

func testResizeOk(t *testing.T, testSvr *httptest.Server) {

	u, err := url.Parse(testSvr.URL + "/resize")
	require.NoError(t, err)
	helperSetQuery(u, "url", testSvr.URL+"/source", "width", "20", "height", "20")

	for _, desc := range []string{
		"from external resource",
		"from cache",
	} {

		res, err := http.Get(u.String())
		require.NoError(t, err, desc)
		require.Equal(t, http.StatusOK, res.StatusCode, desc)

		require.Equal(t, "image/jpeg", res.Header.Get("Content-type"), desc)
		require.Equal(t, "max-age=3600", res.Header.Get("Cache-Control"), desc)

		data, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err, desc)
		require.Equal(t, "4180c34b5cba2a4c3a15a623fd177099", helperMD5(t, data), desc)
	}
}

func helperSetQuery(u *url.URL, arg ...string) {

	if len(arg)%2 != 0 {
		panic("invalid number of arguments")
	}

	q := u.Query()

	for i := 0; i < len(arg)/2; i++ {
		key := arg[i*2]
		value := arg[i*2+1]

		q.Set(key, value)
	}

	u.RawQuery = q.Encode()
}

func helperGetStringFromBody(t *testing.T, res *http.Response) string {

	defer func() {
		require.NoError(t, res.Body.Close())
	}()

	data, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)

	return string(data)
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

func helperMD5(t *testing.T, src []byte) string {
	t.Helper()

	hash := md5.Sum(src)

	buf := make([]byte, len(hash)*2)
	hex.Encode(buf, hash[:])

	return string(buf)
}
