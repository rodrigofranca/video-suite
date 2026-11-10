package httpapi

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"ultramedia/api/internal/app"
	"ultramedia/media-core/domain"
)

// multipartOverhead is the room left for multipart headers and boundaries on
// top of the file size limit.
const multipartOverhead = 1 << 20

// upload handles POST /assets/upload (multipart/form-data, field "file"):
// app.UploadVideo streams the file to storage, without buffering it in
// memory or on disk, and starts the pipeline.
func (a *api) upload(w http.ResponseWriter, r *http.Request) {
	tooLarge := func() {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("file is larger than %d MB", a.MaxUploadBytes>>20))
	}
	_, part, ok := fileField(w, r, a.MaxUploadBytes, tooLarge)
	if !ok {
		return
	}
	id, err := a.uploads.Run(r.Context(), app.UploadRequest{Filename: part.FileName(), Body: part})
	if isMaxBytes(err) {
		tooLarge()
		return
	}
	if err != nil {
		a.useCaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"assetId": id, "status": domain.AssetProcessing})
}

// fileField reads a multipart/form-data body of up to limit bytes of file
// and returns the optional "name" field and the "file" part, which must
// come after it. When it returns false it has answered the request.
func fileField(w http.ResponseWriter, r *http.Request, limit int64, tooLarge func()) (string, *multipart.Part, bool) {
	if r.ContentLength > limit+multipartOverhead {
		tooLarge()
		return "", nil, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit+multipartOverhead)
	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "expected multipart/form-data with a file field")
		return "", nil, false
	}
	name := ""
	for {
		part, err := mr.NextPart()
		switch {
		case isMaxBytes(err):
			tooLarge()
			return "", nil, false
		case err == io.EOF:
			writeError(w, http.StatusBadRequest, "missing file field")
			return "", nil, false
		case err != nil:
			writeError(w, http.StatusBadRequest, err.Error())
			return "", nil, false
		}
		switch part.FormName() {
		case "file":
			return name, part, true
		case "name":
			b, _ := io.ReadAll(io.LimitReader(part, 1024))
			name = string(b)
		}
	}
}

// isMaxBytes reports a request body over its http.MaxBytesReader limit.
func isMaxBytes(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr)
}
