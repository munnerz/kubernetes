/*
Copyright 2014 The Kubernetes Authors All rights reserved.

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

package etcd

import (
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/registry/registrytest"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/storage"
	"k8s.io/kubernetes/pkg/tools"
)

func newStorage(t *testing.T) (*REST, *tools.FakeEtcdClient) {
	etcdStorage, fakeClient := registrytest.NewEtcdStorage(t, "")
	return NewREST(etcdStorage, storage.NoDecoration), fakeClient
}

func validNewLimitRange() *api.LimitRange {
	return &api.LimitRange{
		ObjectMeta: api.ObjectMeta{
			Name:      "foo",
			Namespace: api.NamespaceDefault,
		},
		Spec: api.LimitRangeSpec{
			Limits: []api.LimitRangeItem{
				{
					Type: api.LimitTypePod,
					Max: api.ResourceList{
						api.ResourceCPU:    resource.MustParse("100"),
						api.ResourceMemory: resource.MustParse("10000"),
					},
					Min: api.ResourceList{
						api.ResourceCPU:    resource.MustParse("0"),
						api.ResourceMemory: resource.MustParse("100"),
					},
				},
			},
		},
	}
}

func TestCreate(t *testing.T) {
	storage, fakeClient := newStorage(t)
	test := registrytest.New(t, fakeClient, storage.Etcd).GeneratesName()
	validLimitRange := validNewLimitRange()
	validLimitRange.ObjectMeta = api.ObjectMeta{}
	test.TestCreate(
		// valid
		validLimitRange,
		// invalid
		&api.LimitRange{
			ObjectMeta: api.ObjectMeta{Name: "_-a123-a_"},
		},
	)
}

func TestUpdate(t *testing.T) {
	storage, fakeClient := newStorage(t)
	test := registrytest.New(t, fakeClient, storage.Etcd).AllowCreateOnUpdate()
	test.TestUpdate(
		// valid
		validNewLimitRange(),
		// updateFunc
		func(obj runtime.Object) runtime.Object {
			object := obj.(*api.LimitRange)
			object.Spec.Limits = []api.LimitRangeItem{
				{
					Type: api.LimitTypePod,
					Max: api.ResourceList{
						api.ResourceCPU:    resource.MustParse("1000"),
						api.ResourceMemory: resource.MustParse("100000"),
					},
					Min: api.ResourceList{
						api.ResourceCPU:    resource.MustParse("10"),
						api.ResourceMemory: resource.MustParse("1000"),
					},
				},
			}
			return object
		},
	)
}

func TestDelete(t *testing.T) {
	storage, fakeClient := newStorage(t)
	test := registrytest.New(t, fakeClient, storage.Etcd)
	test.TestDelete(validNewLimitRange())
}

func TestGet(t *testing.T) {
	storage, fakeClient := newStorage(t)
	test := registrytest.New(t, fakeClient, storage.Etcd)
	test.TestGet(validNewLimitRange())
}

func TestList(t *testing.T) {
	storage, fakeClient := newStorage(t)
	test := registrytest.New(t, fakeClient, storage.Etcd)
	test.TestList(validNewLimitRange())
}

func TestWatch(t *testing.T) {
	storage, fakeClient := newStorage(t)
	test := registrytest.New(t, fakeClient, storage.Etcd)
	test.TestWatch(
		validNewLimitRange(),
		// matching labels
		[]labels.Set{},
		// not matching labels
		[]labels.Set{
			{"foo": "bar"},
		},
		// matching fields
		[]fields.Set{},
		// not matching fields
		[]fields.Set{
			{"metadata.name": "bar"},
			{"name": "foo"},
		},
	)
}
