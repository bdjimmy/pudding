package render

import "net/http"

// Render http response render
type Render interface {
	// Render render it to http response writer
	Render(w http.ResponseWriter) error
	// WriteContentType write content-type to http response writer
	WriteContentType(w http.ResponseWriter)
}

// 确认各个输出实现了render 接口
var (
	_ Render = String{}
	_ Render = Redirect{}
	_ Render = JSON{}
	_ Render = MapJSON{}
	_ Render = Data{}
)

func writeContentType(w http.ResponseWriter, value []string){
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = value
	}
}
