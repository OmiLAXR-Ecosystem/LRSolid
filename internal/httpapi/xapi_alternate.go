package httpapi

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// AlternateRequestSyntax rewrites POST ?method=GET|PUT|DELETE form requests
// into the method the xAPI resource actually implements.
func AlternateRequestSyntax(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}
			body, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
			if err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			method := strings.ToUpper(r.URL.Query().Get("method"))
			ct := strings.ToLower(r.Header.Get("Content-Type"))
			var form url.Values
			if strings.Contains(ct, "application/x-www-form-urlencoded") {
				form, _ = url.ParseQuery(string(body))
			} else if strings.Contains(ct, "multipart/form-data") {
				restored := r.Clone(r.Context())
				restored.Body = io.NopCloser(bytes.NewReader(body))
				restored.ContentLength = int64(len(body))
				if err := restored.ParseMultipartForm(32 << 20); err == nil {
					form = restored.Form
				}
			}
			if method == "" && form != nil {
				method = strings.ToUpper(form.Get("method"))
			}
			if method != http.MethodGet && method != http.MethodPut && method != http.MethodDelete {
				r.Body = io.NopCloser(bytes.NewReader(body))
				r.ContentLength = int64(len(body))
				next.ServeHTTP(w, r)
				return
			}

			if form != nil {
				if authz := form.Get("Authorization"); authz != "" && r.Header.Get("Authorization") == "" {
					r.Header.Set("Authorization", authz)
				}
				q := r.URL.Query()
				q.Del("method")
				for key, values := range form {
					if key == "method" || key == "content" || strings.EqualFold(key, "Authorization") || strings.EqualFold(key, "Content-Type") {
						continue
					}
					q.Del(key)
					for _, v := range values {
						q.Add(key, v)
					}
				}
				r.URL.RawQuery = q.Encode()
				if content := form.Get("content"); content != "" {
					body = []byte(content)
					if ctype := form.Get("Content-Type"); ctype != "" {
						r.Header.Set("Content-Type", ctype)
					} else {
						r.Header.Set("Content-Type", "application/json")
					}
				} else {
					body = nil
					r.Header.Del("Content-Type")
				}
			} else {
				q := r.URL.Query()
				q.Del("method")
				r.URL.RawQuery = q.Encode()
			}
			r.Method = method
			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))
			next.ServeHTTP(w, r)
		})
	}(next)
}
