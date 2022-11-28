// Copyright 2022 The Sigstore Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package secret

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	ns   = "test-namespace"
	name = "test-secret"
)

func secret(existing map[string][]byte) *v1.Secret {
	return &v1.Secret{ObjectMeta: meta_v1.ObjectMeta{Namespace: ns, Name: name}, Data: existing}
}

func TestUpdateSecret(t *testing.T) {
	var tests = []struct {
		testName string
		existing map[string][]byte
		in       map[string][]byte
		want     map[string][]byte
	}{
		{
			testName: "non-existing-creates",
			existing: nil,
			in:       map[string][]byte{"foo": []byte("foo-value")},
			want:     map[string][]byte{"foo": []byte("foo-value")},
		},
		{
			testName: "empty-update-no-changes",
			existing: map[string][]byte{"foo": []byte("foo-value")},
			in:       map[string][]byte{},
			want:     map[string][]byte{"foo": []byte("foo-value")},
		},
		{
			testName: "update-field",
			existing: map[string][]byte{"foo": []byte("foo-value")},
			in:       map[string][]byte{"foo": []byte("new-foo")},
			want:     map[string][]byte{"foo": []byte("new-foo")},
		},
		{
			testName: "add-field",
			existing: map[string][]byte{"foo": []byte("foo-value")},
			in:       map[string][]byte{"bar": []byte("bar-value")},
			want:     map[string][]byte{"foo": []byte("foo-value"), "bar": []byte("bar-value")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			objs := []runtime.Object{}
			if tt.existing != nil {
				objs = append(objs, secret(tt.existing))
			}
			client := fake.NewSimpleClientset(objs...)
			err := ReconcileSecret(context.Background(), name, ns, tt.in, client.CoreV1().Secrets(ns))
			if err != nil {
				t.Errorf("Unexpected error updating: %s", err)
				return
			}
			actual, err := client.CoreV1().Secrets(ns).Get(context.Background(), name, meta_v1.GetOptions{})
			if err != nil {
				t.Errorf("Unexpected error getting: %s", err)
				return
			}
			if diff := cmp.Diff(actual.Data, tt.want); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", tt.want, diff)
				return
			}
		})
	}
}
