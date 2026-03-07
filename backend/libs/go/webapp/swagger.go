package webapp

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
)

//go:embed swagger/*
var swaggerUIFS embed.FS

// swaggerSpec represents a discovered swagger JSON specification file.
type swaggerSpec struct {
	// RelPath is the path relative to the specs root, e.g. "api/push/admin.swagger.json".
	RelPath string
	// Name is a human-readable name derived from the file, e.g. "Admin API".
	Name string
}

// registerSwaggerUI sets up HTTP handlers for serving the embedded Swagger UI
// along with dynamically discovered swagger JSON specs from specsDir.
//
// Route layout under /swagger/:
//
//	/swagger/                          → embedded index.html
//	/swagger/swagger-ui.css            → embedded CSS/JS assets
//	/swagger/specs/api/push/admin.swagger.json → spec files from specsDir
func registerSwaggerUI(mux *http.ServeMux, cfg swaggerUIConfig, logger log.Logger) error {
	if !cfg.Enabled {
		return nil
	}

	specsDir := cfg.SpecsDir
	if specsDir == "" {
		specsDir = "./swagger"
	}

	// Discover all .swagger.json files in the specs directory.
	specs, err := discoverSwaggerSpecs(specsDir)
	if err != nil {
		return errors.WrapFailf(err, "discover swagger specs in %v", errors.Token("dir", specsDir))
	}

	if len(specs) == 0 {
		logger.Infof("swagger-ui enabled but no specs found in %v, skipping", errors.Token("dir", specsDir))
		return nil
	}

	logger.Infof("swagger-ui: discovered %v specs in %v", errors.Token("count", len(specs)), errors.Token("dir", specsDir))
	for _, s := range specs {
		logger.Infof("  swagger spec: %v → %v", errors.Token("name", s.Name), errors.Token("path", s.RelPath))
	}

	// Generate the swagger-initializer.js content based on discovered specs.
	initializerJS := generateSwaggerInitializer(specs)

	// Serve the embedded Swagger UI dist files (index.html, CSS, JS, etc.)
	// We strip the "swagger" prefix from the embedded FS so files are at root.
	uiFS, err := fs.Sub(swaggerUIFS, "swagger")
	if err != nil {
		return errors.WrapFail(err, "create sub filesystem for embedded swagger UI")
	}
	uiFileServer := http.FileServer(http.FS(uiFS))

	// Serve the swagger JSON spec files from disk.
	specsFileServer := http.FileServer(http.Dir(specsDir))

	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Override swagger-initializer.js with our dynamically generated version.
		if r.URL.Path == "swagger-initializer.js" {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			_, _ = w.Write([]byte(initializerJS))
			return
		}

		// Serve spec files from the specs/ prefix.
		if strings.HasPrefix(r.URL.Path, "specs/") {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "specs/")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=30, must-revalidate")
			specsFileServer.ServeHTTP(w, r)
			return
		}

		// Everything else comes from the embedded Swagger UI dist.
		w.Header().Set("Cache-Control", "public, max-age=3600, must-revalidate")
		uiFileServer.ServeHTTP(w, r)
	})))

	return nil
}

// discoverSwaggerSpecs walks specsDir recursively and returns all .swagger.json files.
func discoverSwaggerSpecs(specsDir string) ([]swaggerSpec, error) {
	var specs []swaggerSpec

	err := filepath.WalkDir(specsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// If the directory doesn't exist, return empty specs (not an error).
			if os.IsNotExist(err) {
				return filepath.SkipAll
			}
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(d.Name(), ".swagger.json") {
			return nil
		}

		relPath, err := filepath.Rel(specsDir, path)
		if err != nil {
			return errors.WrapFailf(err, "compute relative path for %v", errors.Token("path", path))
		}
		// Normalize to forward slashes for URL paths.
		relPath = filepath.ToSlash(relPath)

		specs = append(specs, swaggerSpec{
			RelPath: relPath,
			Name:    deriveSpecName(relPath),
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return specs, nil
}

func deriveSpecName(relPath string) string {
	name := strings.TrimSuffix(relPath, ".swagger.json")

	parts := strings.Split(name, "/")

	if len(parts) >= 2 {
		pkg := parts[len(parts)-2]
		svc := parts[len(parts)-1]
		return fmt.Sprintf("%s - %s", capitalize(pkg), capitalize(svc))
	}

	if len(parts) == 1 {
		return capitalize(parts[0])
	}

	return relPath
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func generateSwaggerInitializer(specs []swaggerSpec) string {
	var urlEntries []string
	for _, s := range specs {
		urlEntries = append(urlEntries, fmt.Sprintf(
			`      { url: "specs/%s", name: %q }`,
			s.RelPath, s.Name,
		))
	}

	return fmt.Sprintf(`window.onload = function() {
  window.ui = SwaggerUIBundle({
    urls: [
%s
    ],
    dom_id: '#swagger-ui',
    deepLinking: true,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout"
  });
};
`, strings.Join(urlEntries, ",\n"))
}
