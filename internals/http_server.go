package internals

import (
	"io"
	"log"
	"net/http"
)

type writeHandler struct {
	sinks []io.Writer
}

func (wh *writeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range wh.sinks {
		_, err := s.Write(data)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func NewServer() {
	mux := http.NewServeMux()

	wh := &writeHandler{}

	mux.Handle("/stdin", wh)

	http.ListenAndServe(":4000", mux)
}
