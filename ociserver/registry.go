// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package ociregistry implements a docker V2 registry and the OCI distribution specification.
//
// It is designed to be used anywhere a low dependency container registry is needed, with an
// initial focus on tests.
//
// Its goal is to be standards compliant and its strictness will increase over time.
//
// This is currently a low flightmiles system. It's likely quite safe to use in tests; If you're using it
// in production, please let us know how and send us CL's for integration tests.
package ociserver

import (
	"net/http"

	"github.com/rogpeppe/ociregistry"
)

type registry struct {
	blobs            blobs
	manifests        manifests
	referrersEnabled bool
}

// https://docs.docker.com/registry/spec/api/#api-version-check
// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#api-version-check
func (r *registry) v2(resp http.ResponseWriter, req *http.Request) error {
	if isBlob(req) {
		return r.blobs.handle(resp, req)
	}
	if isManifest(req) {
		return r.manifests.handle(resp, req)
	}
	if isTags(req) {
		return r.manifests.handleTags(resp, req)
	}
	if isCatalog(req) {
		return r.manifests.handleCatalog(resp, req)
	}
	if r.referrersEnabled && isReferrers(req) {
		return r.manifests.handleReferrers(resp, req)
	}
	resp.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	if req.URL.Path != "/v2/" && req.URL.Path != "/v2" {
		return ociregistry.NewError("unknown URL path", ociregistry.ErrNameUnknown.Code(), nil)
	}
	resp.WriteHeader(200)
	return nil
}

func (r *registry) root(resp http.ResponseWriter, req *http.Request) {
	if rerr := r.v2(resp, req); rerr != nil {
		writeError(resp, rerr)
		return
	}
}

// Options holds options for the server.
type Options struct {
	// DisableReferrersAPI, when true, causes the registry to behave as if
	// it does not understand the referrers API.
	DisableReferrersAPI bool
}

// New returns a handler which implements the docker registry protocol
// by making calls to the underlying registry backend r.
//
// If opts is nil, it's equivalent to passing new(Options).
//
// The returned handler should be registered at the site root.
func New(backend ociregistry.Interface, opts *Options) http.Handler {
	if opts == nil {
		opts = new(Options)
	}
	r := &registry{
		blobs: blobs{
			backend: backend,
		},
		manifests: manifests{
			backend: backend,
		},
		referrersEnabled: !opts.DisableReferrersAPI,
	}
	return http.HandlerFunc(r.root)
}
