package daemon

import (
	"log"
	"net/http"
)

func (a *App) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stoken := r.Header.Get("X-Auth-Token")
		if stoken == "" || stoken != a.Config.HTTPToken {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(""))
			return
		}
		log.Println(r.RemoteAddr, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
