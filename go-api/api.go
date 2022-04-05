package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

func New(routes map[string]map[string]Handler) Api {
	return api{
		r: routes,
	}
}

type Api interface {
	Serve()
}

type api struct {
	r map[string]map[string]Handler
}

func (api api) Serve() {
	r := mux.NewRouter()
	for path, methods := range api.r {
		for method, handler := range methods {
			r.HandleFunc(path, handler).Methods(method)
		}
	}
	http.Handle("/", r)
	if err := http.ListenAndServe(":3000", nil); err != nil {
		panic(err)
	}
}
