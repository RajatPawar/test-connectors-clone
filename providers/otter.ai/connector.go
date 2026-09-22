// Package otterai hosts the Otter.ai connector.
//
// Otter.ai is a proxy (auth) connector: authenticated raw pass-through to
// Otter.ai's Public API (https://api.otter.ai), with Ampersand handling the
// Bearer API-key auth. It exposes no normalized data model — no Read,
// ListObjectMetadata, or object/field schemas. The entire connector is the
// Provider Catalog entry declared in providers/otter.ai.go (package providers);
// there is intentionally no Connector struct or connector/new.go registration.
//
// This package exists only to satisfy the build gate's ./providers/otter.ai/...
// glob. See providers/otter.ai.go for the catalog entry.
package otterai
