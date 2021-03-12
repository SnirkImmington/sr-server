package routes

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"sr/config"
	"strings"
)

// Serve static files according to webpack's convention.

const hashLength = 8

var validStaticSubdirs = []string{"media", "js", "css"}

func stringArrayContains(array []string, val string) bool {
	for _, inArray := range array {
		if inArray == val {
			return true
		}
	}
	return false
}

func invalidPath(path string) bool {
	trimmed := strings.Trim(path, "/")
	return path == "." || !fs.ValidPath(path)
}

func etagForFile(subFolder string, fileName string) string {
	ext := path.Ext(fileName) // the .map files have the same hash as the originals
	switch subFolder {
	case "css/":
		var lastDot int
		// {main|i}.hash.chunk.css[.map]
		if strings.HasSuffix(fileName, ".map") {
			lastDot := len(fileName) - len(".chunk.css.map")
		} else if strings.HasSuffix(fileName, ".css") {
			lastDot := len(fileName) - len(".chunk.css")
		} else {
			return ""
		}
		// expecting len("main.") or len("i.") leftover
		if len(fileName)-lastDot < 2 {
			return ""
		}
		return fileName[lastDot-hashLength:lastDot] + ext
	case "media/":
		lastDot := strings.LastIndex(fileName, ".")
		// expect at least an original file with a name like f.js
		if lastDot == -1 || len(fileName) < lastDot-hashLength-3 {
			return ""
		}
		return fileName[lastDot-hashLength : lastDot]
	case "js/":
		var offset int
		if strings.HasPrefix(fileName, "runtime-main.") {
			offset = len("runtime-main.")
		} else if strings.HasPrefix(fileName, "main.") {
			offset = len("main.")
		} else {
			offset = strings.Index(fileName, ".")
		}
		if offset == -1 || len(fileName) < offset+hashLength+3 {
			return ""
		}
		return fileName[offset : offset+hashLength]
	default:
		return ""
	}
}

func openFrontendFile(filePath string, useZipped bool, useDefault bool) (*os.File, bool, bool, error) {
	var file *os.File
	var err error
	// Try to open <file>.gz
	if useZipped {
		file, err := os.Open(path.Join(config.FrontendBasePath, filePath+".gz"))
		if err == nil {
			return file, true, false, nil
		}
	}
	// Either not found, or can't use compressed
	// Try to open <file>
	file, err = os.Open(path.Join(config.FrontendBasePath, filePath))
	if err == nil {
		return file, false, false, nil
	}
	// Stop checking here for /static/invalidfile, otherwise default to /index.html
	if !useDefault {
		return nil, false, false, fmt.Errorf("unable to open %v: %v", filePath, err)
	}
	if useZipped {
		// Try to open index.html.gz
		file, err = os.Open(path.Join(config.FrontendBasePath, "index.html.gz"))
		if err == nil {
			return file, false, true, nil
		}
		// allow for not found
		log.Print("Warning: Error opening /index.html.gz with zipping: %v", err)
	}
	// ignore not found, could still be a name issue
	// Try to open index.html
	file, err = os.Open(path.Join(config.FrontendBasePath, "index.html"))
	if err == nil {
		return file, false, true, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, false, true, fmt.Errorf("opening defaulted /index: %w", err)
	}
	// index.html not found
	return nil, false, true, fmt.Errorf("got not found for /index: %w", err)
}

// /static/path
func handleFrontendStatic(response Response, request *Request) {
	logRequest(request)
	requestPath := request.URL.Path
	requestDir, requestFile := path.Split(requestPath)
	fetchGzipped := config.FrontendGzipped && stringArrayContains(request.Header.Values("Content-Type"), "gzip")
	logf(request, "static path = %v, dir = %v, file = %v", requestPath, requestDir, requestFile)

	if invalidPath(requestPath) || strings.Count(requestPath, "/") != 2 {
		logf(request, "Path was invalid")
		httpBadRequest(response, request, requestPath)
	}
	subDir := requestDir[len("static/") : len(requestDir)-1]
	if !stringArrayContains(validStaticSubdirs, subDir) {
		logf(request, "Invalid subdir %v", subDir)
		httpNotFound(response, request, requestPath)
	}
	logf(request, "Received request for static %v file %v", subDir, requestFile)

	// This data is seriously cacheable
	response.Header().Set("Cache-Control", "max-age=31536000, public, immutable")

	etag := etagForFile(subDir, requestFile)
	if etag != "" {
		logf(request, "Adding etag %v", etag)
		response.Header().Add("Etag", etag)
	}

	file, zipped, defaulted, err := openFrontendFile(requestPath, fetchGzipped, false)
	httpInternalErrorIf(response, request, err)
	defer file.Close()

	info, err := file.Stat()
	httpInternalErrorIf(response, request, err)

	if zipped {
		response.Header().Add("Content-Encoding", "gzip")
		response.Header().Add("Vary", "Accept-Encoding")
	}

	if etag == "" {
		logf(request, "!! Unable to find etag for request to %v", requestPath)
	}

	// calls checkIfMatch(), which looks at modtime/If-Unmodified-Since and Etag/If-None-Match header.
	// we've already set the Etag, so if the client's already seen this, we can skip actually reading the file.
	http.ServeContent(response, request, requestFile, info.ModTime(), file)
}

// /path
func handleFrontendBase(response Response, request *Request) {
	logRequest(request)
	requestPath := request.URL.Path
	requestDir, requestFile := path.Split(requestPath)
	fetchGzipped := config.FrontendGzipped && stringArrayContains(request.Header.Values("Content-Type"), "gzip")
	logf(request, "path = `%v`, dir = `%v`, file = `%v`", requestPath, requestDir, requestFile)

	logf(request, "Getting a non-static file")
	resetPath := func() {
		requestDir = "/"
		requestFile = "index.html"
		requestPath = "/index.html"
	}

	// If it's a subdir that's not /static, we know without checking that file doesn't exist
	// This should be the case with most SPA paths, i.e. /join/<gameID> but not /about.
	if strings.Count(requestDir, "/") != 1 {
		logf(request, "It's a subpath, definitely going to need /index.html")
		resetPath()
	}

	// We shouldn't set a max-age cache header, we should rely on if-not-modified (and also just set up PWA stuff).
	// response.Header.Set("Cache-Control", "max-age=86400")

	file, zipped, defaulted, err := openFrontendFile(requestPath, fetchGzipped, true)
	httpInternalErrorIf(response, request, err)
	defer file.Close()

	info, err := file.Stat()
	httpInternalErrorIf(response, request, err)

	if zipped {
		response.Header().Add("Content-Encoding", "gzip")
		response.Header().Add("Vary", "Accept-Encoding")
	}

	// Serve the content in the file, sending the original MIME type (based on file name)
	// calls checkIfMatch(), which only checks modtime since we don't add cache information.
	http.ServeContent(response, request, requestFile, info.ModTime(), file)
}
