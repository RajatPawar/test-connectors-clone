// Package otterai is a placeholder for the Otter.ai connector.
//
// Otter.ai is a proxy (auth) connector: authenticated raw passthrough to
// Otter.ai's Public API. A proxy exposes no normalized data model, so it has
// no Read/ListObjectMetadata code — the whole connector is the Provider Catalog
// entry declared via SetInfo in providers/otter.ai.go (package providers).
//
// This file exists only so the build tooling has a package to compile under
// providers/otter.ai/; it intentionally contains no connector logic.
package otterai
