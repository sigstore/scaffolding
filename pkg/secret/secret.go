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
	"bytes"
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"knative.dev/pkg/logging"
)

// ReconcileSecret takes the name of a secret and a map of what it should
// be like. If the secret is missing it will be created, if it is missing
// some keys, or they are not what is in the map, the secret will be updated.
// If the secret contains other keys, we do not check / delete / update those,
// only ones in the data map are checked and updated.
// nsSecret is a namespaced SecretInterface.
func ReconcileSecret(ctx context.Context, name, ns string, data map[string][]byte, nsSecret v1.SecretInterface) error {
	existingSecret, err := nsSecret.Get(ctx, name, metav1.GetOptions{})
	if err != nil && !apierrs.IsNotFound(err) {
		return fmt.Errorf("failed to get secret %s/%s: %w", ns, name, err)
	}

	// If we found the secret, just make sure all the fields are there.
	if err == nil && existingSecret != nil {
		update := false
		for k := range data {
			if !bytes.Equal(data[k], existingSecret.Data[k]) {
				logging.FromContext(ctx).Infof("secret key %q missing or different than expected, updating", k)
				existingSecret.Data[k] = data[k]
				update = true
			}
		}
		if update {
			_, err = nsSecret.Update(ctx, existingSecret, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update secret %s/%s: %w", ns, name, err)
			}
			logging.FromContext(ctx).Infof("Updated secret %s/%s", ns, name)
		}
	} else {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
			Data: data,
		}
		_, err = nsSecret.Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create secret %s/%s: %w", ns, name, err)
		}
		logging.FromContext(ctx).Infof("Created secret %s/%s", ns, name)
	}
	return nil
}
