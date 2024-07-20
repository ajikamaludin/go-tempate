package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/justinas/alice"
	"github.com/petaki/inertia-go"
)

type App struct {
	Host    string
	Port    string
	Inertia *inertia.Inertia
}

type Manifest map[string]struct {
	File           string   `json:"file"`
	DynamicImports []string `json:"dynamicImports"`
}

func (a *App) Index(w http.ResponseWriter, r *http.Request) {
	a.Inertia.Render(w, r, "Index", map[string]interface{}{
		"data": "index",
	})
}

func (a *App) Dashboard(w http.ResponseWriter, r *http.Request) {
	a.Inertia.Render(w, r, "Dashboard", map[string]interface{}{
		"data": "dashboard",
	})
}

func (a *App) routes(routes map[string]http.HandlerFunc) http.Handler {
	middleware := alice.New(a.Inertia.Middleware)
	mux := http.NewServeMux()

	for route, handler := range routes {
		mux.Handle(route, middleware.ThenFunc(handler))
	}

	mux.Handle("/build/", http.StripPrefix("/build/", http.FileServer(http.Dir("./public/build"))))

	return mux
}

func main() {
	inertia := inertia.New("App", "./public/index.gohtml", "")

	// this is from @vite directive in laravel, need more elegant way to place
	inertia.ShareFunc("vite", func() template.HTML {
		_, err := os.Stat("./public/hot")
		if err == nil {
			return `
			<script type="module">
				import RefreshRuntime from 'http://localhost:5173/@react-refresh'
				RefreshRuntime.injectIntoGlobalHook(window)
				window.$RefreshReg$ = () => {}
				window.$RefreshSig$ = () => (type) => type
				window.__vite_plugin_react_preamble_installed__ = true
			</script>
			<script type="module" src="http://localhost:5173/@vite/client"></script><script type="module" src="http://localhost:5173/resources/js/app.jsx"></script>`
		}

		appjsx, preloads, _ := getPathFromManifest("resources/js/app.jsx")
		var importjsxs string
		for _, x := range preloads {
			importjsx, _, _ := getPathFromManifest(x)
			importjsxs = fmt.Sprintf(`%s<script rel="modulepreload" href="/%s"></script>`, importjsxs, importjsx)
		}
		tmp := template.Must(template.New("test").Parse(fmt.Sprintf(`<script type="module" src="/%s"></script>%s`, appjsx, importjsxs)))

		var buf bytes.Buffer
		tmp.ExecuteTemplate(&buf, "test", nil)

		return template.HTML(buf.String())
	})

	app := &App{
		Host:    "localhost",
		Port:    "3000",
		Inertia: inertia,
	}

	// need a way to define routes of app so the frontend can use it with Ziggy
	routes := map[string]map[string]http.HandlerFunc{
		"index":     {"/": app.Index},
		"dashboard": {"/dashboard": app.Dashboard},
	}

	rr := map[string]http.HandlerFunc{}
	shared := map[string]string{}

	for n, h := range routes {
		for x, v := range h {
			rr[x] = v
			shared[n] = x
		}
	}

	inertia.Share("ziggy", shared)

	addr := fmt.Sprintf("%s:%s", app.Host, app.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: app.routes(rr),
	}

	fmt.Printf("Listen %s\n", addr)
	srv.ListenAndServe()
}

func getPathFromManifest(key string) (string, []string, error) {
	content, err := os.ReadFile("./public/build/manifest.json")
	if err != nil {
		return "", []string{}, fmt.Errorf("could not open file: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return "", []string{}, fmt.Errorf("could not parse JSON: %v", err)
	}

	entry, exists := manifest[key]
	if !exists {
		return "", []string{}, fmt.Errorf("key %s not found in manifest", key)
	}

	return fmt.Sprintf("build/%s", entry.File), entry.DynamicImports, nil
}
