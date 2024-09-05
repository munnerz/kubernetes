/*
Copyright 2015 The Kubernetes Authors.

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

package pods

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/version"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	clientset "k8s.io/client-go/kubernetes"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
	kubeapiservertesting "k8s.io/kubernetes/cmd/kube-apiserver/app/testing"
	"k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/test/integration"
	"k8s.io/kubernetes/test/integration/framework"
)

func TestPodTopologyLabels(t *testing.T) {
	tests := []podTopologyTestCase{
		{
			name: "zone and region topology labels copied from assigned Node",
			targetNodeLabels: map[string]string{
				"topology.k8s.io/zone":   "zone",
				"topology.k8s.io/region": "region",
			},
			expectedPodLabels: map[string]string{
				"topology.k8s.io/zone":   "zone",
				"topology.k8s.io/region": "region",
			},
		},
	}
	// Enable the feature BEFORE starting the test server, as the admission plugin only checks feature gates
	// on start up and not on each invocation at runtime.
	featuregatetesting.SetFeatureGateEmulationVersionDuringTest(t, utilfeature.DefaultFeatureGate, version.MustParse("1.32"))
	featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.SetPodTopologyLabels, true)
	testPodTopologyLabels(t, tests)
}

func TestPodTopologyLabels_FeatureDisabled(t *testing.T) {
	tests := []podTopologyTestCase{
		{
			name: "does nothing when the feature is not enabled",
			targetNodeLabels: map[string]string{
				"topology.k8s.io/zone":   "zone",
				"topology.k8s.io/region": "region",
			},
			expectedPodLabels: map[string]string{},
		},
	}
	// Disable the feature BEFORE starting the test server, as the admission plugin only checks feature gates
	// on start up and not on each invocation at runtime.
	featuregatetesting.SetFeatureGateEmulationVersionDuringTest(t, utilfeature.DefaultFeatureGate, version.MustParse("1.32"))
	featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.SetPodTopologyLabels, false)
	testPodTopologyLabels(t, tests)
}

// podTopologyTestCase is defined outside of TestPodTopologyLabels to allow us to re-use the test implementation logic
// between the feature enabled and feature disabled tests.
// This will no longer be required once the feature gate graduates to GA/locked to being enabled.
type podTopologyTestCase struct {
	name              string
	targetNodeLabels  map[string]string
	expectedPodLabels map[string]string
}

func testPodTopologyLabels(t *testing.T, tests []podTopologyTestCase) {
	server := kubeapiservertesting.StartTestServerOrDie(t, nil, framework.DefaultTestServerFlags(), framework.SharedEtcd())
	defer server.TearDownFn()
	client := clientset.NewForConfigOrDie(server.ClientConfig)
	ns := framework.CreateNamespaceOrDie(client, "pod-topology-labels", t)
	defer framework.DeleteNamespaceOrDie(client, ns, t)

	prototypePod := func() *v1.Pod {
		return &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "pod-topology-test-",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "fake-name",
						Image: "fakeimage",
					},
				},
			},
		}
	}
	prototypeNode := func() *v1.Node {
		return &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "podtopology-test-node-",
			},
		}
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create the Node we are going to bind to.
			node := prototypeNode()
			// Set the labels on the Node we are going to create.
			node.Labels = test.targetNodeLabels
			ctx := context.Background()

			var err error
			if node, err = client.CoreV1().Nodes().Create(ctx, node, metav1.CreateOptions{}); err != nil {
				t.Errorf("Failed to create node: %v", err)
			}

			pod := prototypePod()
			if pod, err = client.CoreV1().Pods(ns.Name).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
				t.Errorf("Failed to create pod: %v", err)
			}

			binding := &v1.Binding{
				ObjectMeta: metav1.ObjectMeta{Name: pod.Name, Namespace: pod.Namespace},
				Target: v1.ObjectReference{
					Kind: "Node",
					Name: node.Name,
				},
			}
			if err := client.CoreV1().Pods(pod.Namespace).Bind(ctx, binding, metav1.CreateOptions{}); err != nil {
				t.Errorf("Failed to bind pod to node: %v", err)
			}

			if pod, err = client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{}); err != nil {
				t.Errorf("Failed to fetch bound Pod: %v", err)
			}

			if !apiequality.Semantic.DeepEqual(pod.Labels, test.expectedPodLabels) {
				t.Errorf("Unexpected label values: %v", cmp.Diff(pod.Labels, test.expectedPodLabels))
			}
		})
	}
}

func TestPodUpdateActiveDeadlineSeconds(t *testing.T) {
	// Disable ServiceAccount admission plugin as we don't have serviceaccount controller running.
	server := kubeapiservertesting.StartTestServerOrDie(t, nil, framework.DefaultTestServerFlags(), framework.SharedEtcd())
	defer server.TearDownFn()

	client := clientset.NewForConfigOrDie(server.ClientConfig)

	ns := framework.CreateNamespaceOrDie(client, "pod-activedeadline-update", t)
	defer framework.DeleteNamespaceOrDie(client, ns, t)

	var (
		iZero = int64(0)
		i30   = int64(30)
		i60   = int64(60)
		iNeg  = int64(-1)
	)

	prototypePod := func() *v1.Pod {
		return &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "xxx",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "fake-name",
						Image: "fakeimage",
					},
				},
			},
		}
	}

	cases := []struct {
		name     string
		original *int64
		update   *int64
		valid    bool
	}{
		{
			name:     "no change, nil",
			original: nil,
			update:   nil,
			valid:    true,
		},
		{
			name:     "no change, set",
			original: &i30,
			update:   &i30,
			valid:    true,
		},
		{
			name:     "change to positive from nil",
			original: nil,
			update:   &i60,
			valid:    true,
		},
		{
			name:     "change to smaller positive",
			original: &i60,
			update:   &i30,
			valid:    true,
		},
		{
			name:     "change to larger positive",
			original: &i30,
			update:   &i60,
			valid:    false,
		},
		{
			name:     "change to negative from positive",
			original: &i30,
			update:   &iNeg,
			valid:    false,
		},
		{
			name:     "change to negative from nil",
			original: nil,
			update:   &iNeg,
			valid:    false,
		},
		// zero is not allowed, must be a positive integer
		{
			name:     "change to zero from positive",
			original: &i30,
			update:   &iZero,
			valid:    false,
		},
		{
			name:     "change to nil from positive",
			original: &i30,
			update:   nil,
			valid:    false,
		},
	}

	for i, tc := range cases {
		pod := prototypePod()
		pod.Spec.ActiveDeadlineSeconds = tc.original
		pod.ObjectMeta.Name = fmt.Sprintf("activedeadlineseconds-test-%v", i)

		if _, err := client.CoreV1().Pods(ns.Name).Create(context.TODO(), pod, metav1.CreateOptions{}); err != nil {
			t.Errorf("Failed to create pod: %v", err)
		}

		pod.Spec.ActiveDeadlineSeconds = tc.update

		_, err := client.CoreV1().Pods(ns.Name).Update(context.TODO(), pod, metav1.UpdateOptions{})
		if tc.valid && err != nil {
			t.Errorf("%v: failed to update pod: %v", tc.name, err)
		} else if !tc.valid && err == nil {
			t.Errorf("%v: unexpected allowed update to pod", tc.name)
		}

		integration.DeletePodOrErrorf(t, client, ns.Name, pod.Name)
	}
}

func TestPodReadOnlyFilesystem(t *testing.T) {
	// Disable ServiceAccount admission plugin as we don't have serviceaccount controller running.
	server := kubeapiservertesting.StartTestServerOrDie(t, nil, framework.DefaultTestServerFlags(), framework.SharedEtcd())
	defer server.TearDownFn()

	client := clientset.NewForConfigOrDie(server.ClientConfig)

	isReadOnly := true
	ns := framework.CreateNamespaceOrDie(client, "pod-readonly-root", t)
	defer framework.DeleteNamespaceOrDie(client, ns, t)

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "xxx",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "fake-name",
					Image: "fakeimage",
					SecurityContext: &v1.SecurityContext{
						ReadOnlyRootFilesystem: &isReadOnly,
					},
				},
			},
		},
	}

	if _, err := client.CoreV1().Pods(ns.Name).Create(context.TODO(), pod, metav1.CreateOptions{}); err != nil {
		t.Errorf("Failed to create pod: %v", err)
	}

	integration.DeletePodOrErrorf(t, client, ns.Name, pod.Name)
}

func TestPodCreateEphemeralContainers(t *testing.T) {
	// Disable ServiceAccount admission plugin as we don't have serviceaccount controller running.
	server := kubeapiservertesting.StartTestServerOrDie(t, nil, framework.DefaultTestServerFlags(), framework.SharedEtcd())
	defer server.TearDownFn()

	client := clientset.NewForConfigOrDie(server.ClientConfig)

	ns := framework.CreateNamespaceOrDie(client, "pod-create-ephemeral-containers", t)
	defer framework.DeleteNamespaceOrDie(client, ns, t)

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "xxx",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:                     "fake-name",
					Image:                    "fakeimage",
					ImagePullPolicy:          "Always",
					TerminationMessagePolicy: "File",
				},
			},
			EphemeralContainers: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
		},
	}

	if _, err := client.CoreV1().Pods(ns.Name).Create(context.TODO(), pod, metav1.CreateOptions{}); err == nil {
		t.Errorf("Unexpected allowed creation of pod with ephemeral containers")
		integration.DeletePodOrErrorf(t, client, ns.Name, pod.Name)
	} else if !strings.HasSuffix(err.Error(), "spec.ephemeralContainers: Forbidden: cannot be set on create") {
		t.Errorf("Unexpected error when creating pod with ephemeral containers: %v", err)
	}
}

// setUpEphemeralContainers creates a pod that has Ephemeral Containers. This is a two step
// process because Ephemeral Containers are not allowed during pod creation.
func setUpEphemeralContainers(podsClient typedv1.PodInterface, pod *v1.Pod, containers []v1.EphemeralContainer) (*v1.Pod, error) {
	result, err := podsClient.Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create pod: %v", err)
	}

	if len(containers) == 0 {
		return result, nil
	}

	pod.Spec.EphemeralContainers = containers
	if _, err := podsClient.Update(context.TODO(), pod, metav1.UpdateOptions{}); err == nil {
		return nil, fmt.Errorf("unexpected allowed direct update of ephemeral containers during set up: %v", err)
	}

	result, err = podsClient.UpdateEphemeralContainers(context.TODO(), pod.Name, pod, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update ephemeral containers for test case set up: %v", err)
	}

	return result, nil
}

func TestPodPatchEphemeralContainers(t *testing.T) {
	// Disable ServiceAccount admission plugin as we don't have serviceaccount controller running.
	server := kubeapiservertesting.StartTestServerOrDie(t, nil, framework.DefaultTestServerFlags(), framework.SharedEtcd())
	defer server.TearDownFn()

	client := clientset.NewForConfigOrDie(server.ClientConfig)

	ns := framework.CreateNamespaceOrDie(client, "pod-patch-ephemeral-containers", t)
	defer framework.DeleteNamespaceOrDie(client, ns, t)

	testPod := func(name string) *v1.Pod {
		return &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:                     "fake-name",
						Image:                    "fakeimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
		}
	}

	cases := []struct {
		name      string
		original  []v1.EphemeralContainer
		patchType types.PatchType
		patchBody []byte
		valid     bool
	}{
		{
			name:      "create single container (strategic)",
			original:  nil,
			patchType: types.StrategicMergePatchType,
			patchBody: []byte(`{
				"spec": {
					"ephemeralContainers": [{
						"name": "debugger1",
						"image": "debugimage",
						"imagePullPolicy": "Always",
						"terminationMessagePolicy": "File"
					}]
				}
			}`),
			valid: true,
		},
		{
			name:      "create single container (merge)",
			original:  nil,
			patchType: types.MergePatchType,
			patchBody: []byte(`{
				"spec": {
					"ephemeralContainers":[{
						"name": "debugger1",
						"image": "debugimage",
						"imagePullPolicy": "Always",
						"terminationMessagePolicy": "File"
					}]
				}
			}`),
			valid: true,
		},
		{
			name:      "create single container (JSON)",
			original:  nil,
			patchType: types.JSONPatchType,
			// Because ephemeralContainers is optional, a JSON patch of an empty ephemeralContainers must add the
			// list rather than simply appending to it.
			patchBody: []byte(`[{
				"op":"add",
				"path":"/spec/ephemeralContainers",
				"value":[{
					"name":"debugger1",
					"image":"debugimage",
					"imagePullPolicy": "Always",
					"terminationMessagePolicy": "File"
				}]
			}]`),
			valid: true,
		},
		{
			name: "add single container (strategic)",
			original: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger1",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			patchType: types.StrategicMergePatchType,
			patchBody: []byte(`{
				"spec": {
					"ephemeralContainers":[{
						"name": "debugger2",
						"image": "debugimage",
						"imagePullPolicy": "Always",
						"terminationMessagePolicy": "File"
					}]
				}
			}`),
			valid: true,
		},
		{
			name: "add single container (merge)",
			original: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger1",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			patchType: types.MergePatchType,
			patchBody: []byte(`{
				"spec": {
					"ephemeralContainers":[{
						"name": "debugger1",
						"image": "debugimage",
						"imagePullPolicy": "Always",
						"terminationMessagePolicy": "File"
					},{
						"name": "debugger2",
						"image": "debugimage",
						"imagePullPolicy": "Always",
						"terminationMessagePolicy": "File"
					}]
				} 
			}`),
			valid: true,
		},
		{
			name: "add single container (JSON)",
			original: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger1",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			patchType: types.JSONPatchType,
			patchBody: []byte(`[{
				"op":"add",
				"path":"/spec/ephemeralContainers/-",
				"value":{
					"name":"debugger2",
					"image":"debugimage",
					"imagePullPolicy": "Always",
					"terminationMessagePolicy": "File"
				}
			}]`),
			valid: true,
		},
		{
			name: "remove all containers (merge)",
			original: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger1",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			patchType: types.MergePatchType,
			patchBody: []byte(`{"spec": {"ephemeralContainers":[]}}`),
			valid:     false,
		},
		{
			name: "remove the single container (JSON)",
			original: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger1",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			patchType: types.JSONPatchType,
			patchBody: []byte(`[{"op":"remove","path":"/spec/ephemeralContainers/0"}]`),
			valid:     false, // disallowed by policy rather than patch semantics
		},
		{
			name: "remove all containers (JSON)",
			original: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger1",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			patchType: types.JSONPatchType,
			patchBody: []byte(`[{"op":"remove","path":"/spec/ephemeralContainers"}]`),
			valid:     false, // disallowed by policy rather than patch semantics
		},
	}

	for i, tc := range cases {
		pod := testPod(fmt.Sprintf("ephemeral-container-test-%v", i))
		if _, err := setUpEphemeralContainers(client.CoreV1().Pods(ns.Name), pod, tc.original); err != nil {
			t.Errorf("%v: %v", tc.name, err)
		}

		if _, err := client.CoreV1().Pods(ns.Name).Patch(context.TODO(), pod.Name, tc.patchType, tc.patchBody, metav1.PatchOptions{}, "ephemeralcontainers"); tc.valid && err != nil {
			t.Errorf("%v: failed to update ephemeral containers: %v", tc.name, err)
		} else if !tc.valid && err == nil {
			t.Errorf("%v: unexpected allowed update to ephemeral containers", tc.name)
		}

		integration.DeletePodOrErrorf(t, client, ns.Name, pod.Name)
	}
}

func TestPodUpdateEphemeralContainers(t *testing.T) {
	// Disable ServiceAccount admission plugin as we don't have serviceaccount controller running.
	server := kubeapiservertesting.StartTestServerOrDie(t, nil, framework.DefaultTestServerFlags(), framework.SharedEtcd())
	defer server.TearDownFn()

	client := clientset.NewForConfigOrDie(server.ClientConfig)

	ns := framework.CreateNamespaceOrDie(client, "pod-update-ephemeral-containers", t)
	defer framework.DeleteNamespaceOrDie(client, ns, t)

	testPod := func(name string) *v1.Pod {
		return &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "fake-name",
						Image: "fakeimage",
					},
				},
			},
		}
	}

	cases := []struct {
		name     string
		original []v1.EphemeralContainer
		update   []v1.EphemeralContainer
		valid    bool
	}{
		{
			name:     "no change, nil",
			original: nil,
			update:   nil,
			valid:    true,
		},
		{
			name: "no change, set",
			original: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			update: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			valid: true,
		},
		{
			name:     "add single container",
			original: nil,
			update: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			valid: true,
		},
		{
			name: "remove all containers, nil",
			original: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			update: nil,
			valid:  false,
		},
		{
			name: "remove all containers, empty",
			original: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			update: []v1.EphemeralContainer{},
			valid:  false,
		},
		{
			name: "increase number of containers",
			original: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger1",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			update: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger1",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger2",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			valid: true,
		},
		{
			name: "decrease number of containers",
			original: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger1",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger2",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			update: []v1.EphemeralContainer{
				{
					EphemeralContainerCommon: v1.EphemeralContainerCommon{
						Name:                     "debugger1",
						Image:                    "debugimage",
						ImagePullPolicy:          "Always",
						TerminationMessagePolicy: "File",
					},
				},
			},
			valid: false,
		},
	}

	for i, tc := range cases {
		pod, err := setUpEphemeralContainers(client.CoreV1().Pods(ns.Name), testPod(fmt.Sprintf("ephemeral-container-test-%v", i)), tc.original)
		if err != nil {
			t.Errorf("%v: %v", tc.name, err)
		}

		pod.Spec.EphemeralContainers = tc.update
		if _, err := client.CoreV1().Pods(ns.Name).UpdateEphemeralContainers(context.TODO(), pod.Name, pod, metav1.UpdateOptions{}); tc.valid && err != nil {
			t.Errorf("%v: failed to update ephemeral containers: %v", tc.name, err)
		} else if !tc.valid && err == nil {
			t.Errorf("%v: unexpected allowed update to ephemeral containers", tc.name)
		}

		integration.DeletePodOrErrorf(t, client, ns.Name, pod.Name)
	}
}

func TestMutablePodSchedulingDirectives(t *testing.T) {
	// Disable ServiceAccount admission plugin as we don't have serviceaccount controller running.
	server := kubeapiservertesting.StartTestServerOrDie(t, nil, framework.DefaultTestServerFlags(), framework.SharedEtcd())
	defer server.TearDownFn()

	client := clientset.NewForConfigOrDie(server.ClientConfig)

	ns := framework.CreateNamespaceOrDie(client, "mutable-pod-scheduling-directives", t)
	defer framework.DeleteNamespaceOrDie(client, ns, t)

	cases := []struct {
		name   string
		create *v1.Pod
		update *v1.Pod
		err    string
	}{
		{
			name: "adding node selector is allowed for gated pods",
			create: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "fake-name",
							Image: "fakeimage",
						},
					},
					SchedulingGates: []v1.PodSchedulingGate{{Name: "baz"}},
				},
			},
			update: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "fake-name",
							Image: "fakeimage",
						},
					},
					NodeSelector: map[string]string{
						"foo": "bar",
					},
					SchedulingGates: []v1.PodSchedulingGate{{Name: "baz"}},
				},
			},
		},
		{
			name: "addition to nodeAffinity is allowed for gated pods",
			create: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "fake-name",
							Image: "fakeimage",
						},
					},
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "expr",
												Operator: v1.NodeSelectorOpIn,
												Values:   []string{"foo"},
											},
										},
									},
								},
							},
						},
					},
					SchedulingGates: []v1.PodSchedulingGate{{Name: "baz"}},
				},
			},
			update: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "fake-name",
							Image: "fakeimage",
						},
					},
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
								// Add 1 MatchExpression and 1 MatchField.
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "expr",
												Operator: v1.NodeSelectorOpIn,
												Values:   []string{"foo"},
											},
											{
												Key:      "expr2",
												Operator: v1.NodeSelectorOpIn,
												Values:   []string{"foo2"},
											},
										},
										MatchFields: []v1.NodeSelectorRequirement{
											{
												Key:      "metadata.name",
												Operator: v1.NodeSelectorOpIn,
												Values:   []string{"foo"},
											},
										},
									},
								},
							},
						},
					},
					SchedulingGates: []v1.PodSchedulingGate{{Name: "baz"}},
				},
			},
		},
		{
			name: "addition to nodeAffinity is allowed for gated pods with nil affinity",
			create: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "fake-name",
							Image: "fakeimage",
						},
					},
					SchedulingGates: []v1.PodSchedulingGate{{Name: "baz"}},
				},
			},
			update: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "fake-name",
							Image: "fakeimage",
						},
					},
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
								// Add 1 MatchExpression and 1 MatchField.
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "expr",
												Operator: v1.NodeSelectorOpIn,
												Values:   []string{"foo"},
											},
										},
										MatchFields: []v1.NodeSelectorRequirement{
											{
												Key:      "metadata.name",
												Operator: v1.NodeSelectorOpIn,
												Values:   []string{"foo"},
											},
										},
									},
								},
							},
						},
					},
					SchedulingGates: []v1.PodSchedulingGate{{Name: "baz"}},
				},
			},
		},
	}
	for _, tc := range cases {
		if _, err := client.CoreV1().Pods(ns.Name).Create(context.TODO(), tc.create, metav1.CreateOptions{}); err != nil {
			t.Errorf("Failed to create pod: %v", err)
		}

		_, err := client.CoreV1().Pods(ns.Name).Update(context.TODO(), tc.update, metav1.UpdateOptions{})
		if (tc.err == "" && err != nil) || (tc.err != "" && err != nil && !strings.Contains(err.Error(), tc.err)) {
			t.Errorf("Unexpected error: got %q, want %q", err.Error(), err)
		}
		integration.DeletePodOrErrorf(t, client, ns.Name, tc.update.Name)
	}
}
