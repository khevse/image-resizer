package images

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/khevse/image-resizer/service/images/internal/buffer"
	"github.com/khevse/image-resizer/service/images/internal/cache"
	"github.com/khevse/image-resizer/service/images/internal/picture"
)

// Handler images server mux object
type Handler struct {
	cache         *cache.Cache
	maxFileSize   int64
	cacheLifetime string
}

// New images handler
func New() *Handler {

	const MB int64 = 1024 * 1024 * 1024

	cacheLifetime := time.Hour

	return &Handler{
		maxFileSize:   MB,
		cache:         cache.New(MB, 50, cacheLifetime, time.Second),
		cacheLifetime: "max-age=" + strconv.FormatInt(int64(cacheLifetime.Seconds()), 10),
	}
}

// Close internal cache
func (h *Handler) Close() error {
	return h.cache.Close()
}

// Mux returns server mux of images server
func (h *Handler) Mux() *http.ServeMux {

	mux := http.NewServeMux()
	mux.HandleFunc("/resize", h.Resize)

	return mux
}

// Resize image
func (h *Handler) Resize(w http.ResponseWriter, req *http.Request) {

	q := req.URL.Query()

	reqURL := q.Get("url")
	reqWidthStr := q.Get("width")
	reqHeightStr := q.Get("height")

	if reqURL == "" {
		http.Error(w, "invalid resource URL", 400)
		return
	}

	reqWidth, err := strconv.Atoi(reqWidthStr)
	if err != nil {
		http.Error(w, "invalid property width: "+err.Error(), 400)
		return
	} else if reqWidth <= 0 {
		http.Error(w, "invalid property width", 400)
		return
	}

	reqHeight, err := strconv.Atoi(reqHeightStr)
	if err != nil {
		http.Error(w, "invalid property height: "+err.Error(), 400)
		return
	} else if reqHeight <= 0 {
		http.Error(w, "invalid property height", 400)
		return
	}

	w.Header().Add("Cache-Control", h.cacheLifetime)

	cacheKey := cache.NewKey(reqURL + reqWidthStr + reqHeightStr)
	cacheVal, ok := h.cache.Get(cacheKey)
	if ok {
		log.Println("send image from cache")

		_, err := io.Copy(w, bytes.NewReader(cacheVal))
		if err != nil {
			http.Error(w, "failed to send request:"+err.Error(), 500)
			return
		}

	} else {
		log.Println("send from resource")

		reqForLoad, err := http.NewRequest(http.MethodGet, reqURL, nil)
		if err != nil {
			http.Error(w, "failed to create request:"+err.Error(), 400)
			return
		}

		res, err := http.DefaultClient.Do(reqForLoad)
		if err != nil {
			http.Error(w, "failed to send request:"+err.Error(), 400)
			return
		}

		defer func() {
			if err := res.Body.Close(); err != nil {
				log.Println("ERROR:", err)
			}
		}()

		srcReader := bufio.NewReaderSize(res.Body, 512)

		buf := buffer.New(int(atomic.LoadInt64(&h.maxFileSize)))
		wr := io.MultiWriter(buf, w)
		if err := picture.Resize(wr, srcReader, uint(reqWidth), uint(reqHeight)); err != nil {
			http.Error(w, "internal server error:"+err.Error(), 500)
			return
		}

		if data, ok := buf.Get(); ok {
			h.cache.Add(cacheKey, data)
		}
	}

	w.Header().Add("Content-type", "image/jpeg")
}
