// pprof middleware
package perf

import (
	"sync"
	"net/http"
	"net/http/pprof"
	"github.com/pkg/errors"
)

var (
	_perfOnce sync.Once
)

func StartPerf(){
	_perfOnce.Do(func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

		go func() {
			if err := http.ListenAndServe("0.0.0.0:2333", mux); err != nil{
				panic(errors.Errorf("pudding: listen %s: error(%v)", "0.0.0.0:2333", err))
			}
		}()
	})
}












