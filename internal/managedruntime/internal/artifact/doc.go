// Package artifact verifies pinned downloads and publishes isolated runtime
// bundles atomically. It validates archive layouts and filesystem ownership;
// provider selection, authorization, and process lifetime belong to its caller.
package artifact
