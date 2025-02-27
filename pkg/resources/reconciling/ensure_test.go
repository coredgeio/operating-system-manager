/*
Copyright 2021 The Operating System Manager contributors.

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

package reconciling

import (
	"context"
	"testing"

	"github.com/go-test/deep"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	controllerruntimefake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureObjectByAnnotation(t *testing.T) {
	const (
		testNamespace    = "default"
		testResourceName = "test"
	)

	tests := []struct {
		name           string
		creator        ObjectCreator
		existingObject controllerruntimeclient.Object
		expectedObject controllerruntimeclient.Object
	}{
		{
			name: "Object gets created",
			expectedObject: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            testResourceName,
					Namespace:       testNamespace,
					ResourceVersion: "1",
				},
				Data: map[string]string{
					"foo": "bar",
				},
			},
			creator: func(existing controllerruntimeclient.Object) (controllerruntimeclient.Object, error) {
				var sa *corev1.ConfigMap
				if existing == nil {
					sa = &corev1.ConfigMap{}
				} else {
					sa = existing.(*corev1.ConfigMap)
				}
				sa.Name = testResourceName
				sa.Namespace = testNamespace
				sa.Data = map[string]string{
					"foo": "bar",
				}
				return sa, nil
			},
		},
		{
			name: "Object gets updated",
			existingObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            testResourceName,
					Namespace:       testNamespace,
					ResourceVersion: "0",
				},
				Data: map[string]string{
					"foo": "hopefully-gets-overwritten",
				},
			},
			creator: func(existing controllerruntimeclient.Object) (controllerruntimeclient.Object, error) {
				var sa *corev1.ConfigMap
				if existing == nil {
					sa = &corev1.ConfigMap{}
				} else {
					sa = existing.(*corev1.ConfigMap)
				}
				sa.Name = testResourceName
				sa.Namespace = testNamespace
				sa.Data = map[string]string{
					"foo": "bar",
				}
				return sa, nil
			},
			expectedObject: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            testResourceName,
					Namespace:       testNamespace,
					ResourceVersion: "1",
				},
				Data: map[string]string{
					"foo": "bar",
				},
			},
		},
		{
			name: "Object does not get updated",
			existingObject: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            testResourceName,
					Namespace:       testNamespace,
					ResourceVersion: "0",
				},
				Data: map[string]string{
					"foo": "bar",
				},
			},
			creator: func(existing controllerruntimeclient.Object) (controllerruntimeclient.Object, error) {
				var sa *corev1.ConfigMap
				if existing == nil {
					sa = &corev1.ConfigMap{}
				} else {
					sa = existing.(*corev1.ConfigMap)
				}
				sa.Name = testResourceName
				sa.Namespace = testNamespace
				sa.Data = map[string]string{
					"foo": "bar",
				}
				return sa, nil
			},
			expectedObject: &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            testResourceName,
					Namespace:       testNamespace,
					ResourceVersion: "0",
				},
				Data: map[string]string{
					"foo": "bar",
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			var client controllerruntimeclient.Client
			if test.existingObject != nil {
				client = controllerruntimefake.
					NewClientBuilder().
					WithScheme(scheme.Scheme).
					WithObjects(test.existingObject).
					Build()
			} else {
				client = controllerruntimefake.
					NewClientBuilder().
					WithScheme(scheme.Scheme).
					Build()
			}

			name := types.NamespacedName{Namespace: testNamespace, Name: testResourceName}
			if err := EnsureNamedObject(context.Background(), name, test.creator, client, &corev1.ConfigMap{}, false); err != nil {
				t.Errorf("EnsureObject returned an error while none was expected: %v", err)
			}

			key := controllerruntimeclient.ObjectKeyFromObject(test.expectedObject)

			gotConfigMap := &corev1.ConfigMap{}
			if err := client.Get(context.Background(), key, gotConfigMap); err != nil {
				t.Fatalf("Failed to get the ConfigMap from the client: %v", err)
			}

			if diff := deep.Equal(gotConfigMap, test.expectedObject); diff != nil {
				t.Errorf("configMap from the client does not match the expected ConfigMap. Diff: \n%v", diff)
			}
		})
	}
}
