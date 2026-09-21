// Package otterai holds Otter.ai connector code.
//
// Otter.ai is currently a proxy (auth) connector: it provides managed,
// authenticated raw pass-through to Otter.ai's Public API and exposes no
// normalized data model (no Read/ListObjectMetadata). The entire proxy
// connector is the Provider Catalog entry declared in providers/otter.ai.go
// (AuthType ApiKey, "Authorization: Bearer <API_KEY>", BaseURL
// https://api.otter.ai).
//
// If a deep read/write connector is layered on later, its implementation
// (connector.go, supports.go, handlers.go, parse.go) lives in this package.
package otterai
