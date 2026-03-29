package nginx

import "embed"

//go:embed templates/*.tmpl
var templateFS embed.FS
