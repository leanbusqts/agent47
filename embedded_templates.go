package agent47embed

import "embed"

// TemplatesFS embeds the shipped template payload for release builds.
//
//go:embed templates templates/* templates/**
var TemplatesFS embed.FS
