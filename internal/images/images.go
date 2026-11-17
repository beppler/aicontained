// Package images embeds the container image definitions, so the compiled
// binary can build the images without a repo checkout.
package images

import "embed"

//go:embed containers
var Containers embed.FS
