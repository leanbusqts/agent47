package agent47embed

import "embed"

// TemplatesFS embeds the shipped template payload for release builds.
//
//go:embed templates templates/* templates/** templates/base/.agents templates/base/.agents/specs templates/base/.agents/specs/spec.yml
var TemplatesFS embed.FS
