package render

import (
	"net/http"
	"encoding/json"
	"github.com/pkg/errors"
)

var jsonContentType = []string{"application/json; charset=utf-8"}

type JSON struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	TTL     int         `json:"ttl"`
	Data    interface{} `json:"data, omitempty"`
}

// Render writes data with json ContentType
func (r JSON)Render(w http.ResponseWriter) error{
	if r.TTL <= 0 {
		r.TTL = 1
	}
	return writeJSON(w, r)
}

func (r JSON) WriteContentType(w http.ResponseWriter){
	writeContentType(w, jsonContentType)
}

func writeJSON(w http.ResponseWriter, obj interface{})(err error){
	var jsonBytes []byte
	writeContentType(w, jsonContentType)
	if jsonBytes, err = json.Marshal(obj); err != nil{
		err = errors.WithStack(err)
		return
	}
	if _, err = w.Write(jsonBytes); err != nil{
		err = errors.WithStack(err)
	}
	return
}


// MapJSON common map json struct
type MapJSON map[string]interface{}

func(m MapJSON) Render(w http.ResponseWriter) error{
	return writeJSON(w, m)
}

func (m MapJSON)WriteContentType(w http.ResponseWriter){
	writeContentType(w, jsonContentType)
}