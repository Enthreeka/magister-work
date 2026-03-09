// Package tmplsrc holds embedded template sources for the Go language plugin.
// It is a leaf package with no internal dependencies to avoid import cycles.
package tmplsrc

import _ "embed"

//go:embed domain.go.tmpl
var DomainTemplate string

//go:embed service.go.tmpl
var ServiceTemplate string

//go:embed handler_gin.go.tmpl
var HandlerGinTemplate string
