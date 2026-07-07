// Package gotemplates embeds the IEQ html/css/js chart templates so binaries
// can be run from any working directory.
package gotemplates

import "embed"

// FS holds all template files, addressed relative to this directory,
// e.g. "common/metrics.html".
//
//go:embed common iaq ieq ieqcharts ieqdonut
var FS embed.FS
