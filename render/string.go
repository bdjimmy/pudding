package render

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
)

var plainContentType = []string{"text/plain; charset=utf-8"}

// String common string struct
type String struct {
	Format string
	Data   []interface{}
}

func (r String) Render(w http.ResponseWriter) error {
	return writeString(w, r.Format, r.Data)
}

func (r String) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, plainContentType)
}

func writeString(w http.ResponseWriter, format string, data []interface{}) (err error) {
	writeContentType(w, plainContentType)
	if len(data) > 0 {
		_, err = fmt.Fprintf(w, format, data...)
	} else {
		_, err = io.WriteString(w, format)
	}
	if err != nil {
		err = errors.WithStack(err)
	}
	return
}
