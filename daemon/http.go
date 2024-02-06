package daemon

import (
	"log"
	"net/http"
)

// AuthMiddleware is a middleware to check if the request has a valid token
func (a *App) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stoken := r.Header.Get("X-Auth-Token")
		if stoken == "" || stoken != a.Config.HTTPToken {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(""))
			if err != nil {
				log.Println("AuthMiddleware", err)
			}
			return
		}
		log.Println(r.RemoteAddr, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
