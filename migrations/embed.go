package migrations

import "embed"

// Files contains the ordered SQL migrations shipped with the Control Plane.
//
//go:embed *.sql
var Files embed.FS
