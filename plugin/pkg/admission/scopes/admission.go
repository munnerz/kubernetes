/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package scopes contains an admission controller for automatically setting the status.scopeID field when
// the status.namespaces field changes on ScopeDefinition objects.
package scopes

import (
	"context"
	"io"
	"sort"
	"strings"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/kubernetes/pkg/apis/scopes"
)

// PluginName indicates name of admission plugin.
const PluginName = "ScopeDefinition"

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewScopeDefinition(), nil
	})
}

// ScopeDefinition is an implementation of admission.Interface.
// When an UPDATE is submitted to the scopedefinitions/status endpoint, it ensures the status.scopeID is updated
// whenever the list of namespaces in status.namespaces has changed.
type ScopeDefinition struct {
	*admission.Handler

	versioner storage.Versioner
}

var _ admission.MutationInterface = &ScopeDefinition{}

// Admit makes an admission decision based on the request attributes
func (r *ScopeDefinition) Admit(ctx context.Context, attributes admission.Attributes, o admission.ObjectInterfaces) error {
	// Only operate on the 'scopedefinitions/status' subresource.
	if attributes.GetResource().Group != scopes.GroupName ||
		attributes.GetResource().Resource != "scopedefinitions" ||
		attributes.GetSubresource() != "status" {
		return nil
	}
	oldObj, _ := attributes.GetOldObject().(*scopes.ScopeDefinition)
	obj, _ := attributes.GetObject().(*scopes.ScopeDefinition)

	// we must update status.scopeID if either the oldObj is not set (create on update on status endpoint?) or if the
	// status.namespaces list has changed.
	requiresNewScopeID := oldObj == nil || (!apiequality.Semantic.DeepEqual(oldObj.Status.Namespaces, obj.Status.Namespaces))
	if requiresNewScopeID {
		// we don't allow users to explicitly set the scopeID ever, so always override it at this point
		obj.Status.ScopeID = string(uuid.NewUUID())
	}
	// compute the new minimumResourceVersions
	var err error
	if obj.Status.MinimumResourceVersions, err = buildMinimumResourceVersions(r.versioner, obj.Status.ServerScopeVersions); err != nil {
		return err
	}
	return nil
}

func buildMinimumResourceVersions(versioner storage.Versioner, ssvs []scopes.ServerScopeVersion) ([]scopes.MinimumResourceVersion, error) {
	type val struct {
		scopes.ServerScopeVersion
		rv uint64
	}
	groupedByStore := make(map[string][]val)
	for _, ssv := range ssvs {
		rv, err := versioner.ParseResourceVersion(ssv.ResourceVersion)
		if err != nil {
			return nil, err
		}
		groupedByStore[ssv.StoreID] = append(groupedByStore[ssv.StoreID], val{ssv, rv})
	}
	var minimumResourceVersions []scopes.MinimumResourceVersion
	for storeID, vals := range groupedByStore {
		sort.Slice(vals, func(i, j int) bool {
			return vals[i].rv > vals[j].rv
		})
		minimumResourceVersions = append(minimumResourceVersions, scopes.MinimumResourceVersion{
			StoreID:         storeID,
			ResourceVersion: vals[0].ResourceVersion,
		})
	}
	sort.SliceStable(minimumResourceVersions, func(i, j int) bool {
		return strings.Compare(minimumResourceVersions[i].StoreID, minimumResourceVersions[j].StoreID) < 0
	})
	return minimumResourceVersions, nil
}

// NewScopeDefinition creates a new ScopeDefinition admission control handler
func NewScopeDefinition() *ScopeDefinition {
	return &ScopeDefinition{
		versioner: storage.APIObjectVersioner{},
		Handler:   admission.NewHandler(admission.Update),
	}
}
