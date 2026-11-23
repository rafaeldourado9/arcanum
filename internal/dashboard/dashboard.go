// Package dashboard serve o painel web (React, compilado em dist/ pelo `npm run build` da
// pasta web/) de dentro do proprio binario: abriu o gateway, o painel esta em /.
package dashboard

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var files embed.FS

// Handler serve os arquivos do painel; caminho que nao e arquivo devolve o index.html
// (rotas do React Router). Sem build do painel, explica como gerar.
func Handler() http.Handler {
	dist, _ := fs.Sub(files, "dist")
	index, err := fs.ReadFile(dist, "index.html")
	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Arcanum dashboard is not built into this binary.\nBuild it with: cd web && npm ci && npm run build\nThe API is available at /api and the docs at /docs.\n"))
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" && name != "index.html" {
			if st, statErr := fs.Stat(dist, name); statErr == nil && !st.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					// nomes com hash do Vite: podem ficar em cache para sempre
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(index)
	})
}
