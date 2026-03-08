// Package conduitregistry loads and stores SQL migration files.
//
// Use [FromFS] to parse migration files from a filesystem, then pass the
// resulting [Registry] to [conduit.WithRegistry].
package conduitregistry
