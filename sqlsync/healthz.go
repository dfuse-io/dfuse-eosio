package sqlsync

import "net/http"

func (s *SQLSync) HealthzHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if false {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("not ready"))
			return
		}
		w.Write([]byte("ok"))
		return
	})
}
