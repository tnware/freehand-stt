// Package managedruntime owns optional isolated speech and cleanup runtimes.
// Manager coordinates inventory publication and admission; each worker owns one
// instance's operations, cancellation, endpoint leases, and bounded private output.
// Provider adapters qualify models and choose pinned platform recipes. Private
// artifact and process packages own verified installation and native child trees.
// Runtime configuration never reads manual endpoints or stored credentials.
package managedruntime
