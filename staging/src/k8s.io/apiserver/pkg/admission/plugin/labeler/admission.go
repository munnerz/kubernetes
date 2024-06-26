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

// Package labeler contains an admission plugin that sets the k8s.io/namespace label on all namespaced objects.
package labeler

import (
	"context"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apiserver/pkg/admission"
	// install the clientgo image policy API for use with api registry
	_ "k8s.io/kubernetes/pkg/apis/imagepolicy/install"
)

// PluginName indicates name of admission plugin.
const PluginName = "NamespaceLabeler"

const (
	// NamespaceNameLabelKey is the key of the label set on all objects containing their namespace.
	NamespaceNameLabelKey string = "kubernetes.io/metadata.namespace"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(_ io.Reader) (admission.Interface, error) {
		return &Plugin{
			Handler: admission.NewHandler(admission.Create, admission.Update),
		}, nil
	})
}

// Plugin is an implementation of admission.Interface.
type Plugin struct {
	*admission.Handler
}

var _ admission.MutationInterface = &Plugin{}

func (p *Plugin) Admit(ctx context.Context, attributes admission.Attributes, o admission.ObjectInterfaces) (err error) {
	obj, err := meta.Accessor(attributes.GetObject())
	if err != nil {
		return fmt.Errorf("object does not implement meta accessor: %w", err)
	}
	// only modify namespace-scoped objects
	if obj.GetNamespace() == "" {
		return nil
	}

	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels[NamespaceNameLabelKey] = obj.GetNamespace()
	obj.SetLabels(labels)
	return nil
}
