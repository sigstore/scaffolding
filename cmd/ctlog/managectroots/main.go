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

package main

import (
	"flag"
	"net/url"
	"os"
	"strings"

	fulcioclient "github.com/sigstore/fulcio/pkg/api"
	"github.com/sigstore/scaffolding/pkg/ctlog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
	"sigs.k8s.io/release-utils/version"
)

const (
	// Key in the configmap holding the value of the tree.
	treeKey   = "treeID"
	configKey = "config"
	bitSize   = 4096
)

var (
	cmname         = flag.String("configmap", "ctlog-config", "Name of the configmap where the treeID lives. if configInSecret is false, ctlog config gets added here also.")
	configInSecret = flag.Bool("config-in-secret", false, "If set to true, fetch / update the ctlog configuration proto into a secret specified in ctlog-secrets under key 'config'.")
	secretName     = flag.String("secret", "ctlog-secrets", "Name of the secret to fetch private key for CTLog.")
	fulcioURL      = flag.String("fulcio-url", "http://fulcio.fulcio-system.svc", "Where to fetch the fulcio Root CA from.")
	operation      = flag.String("operation", "", "Operation to perform for the specified fulcio [add,remove]")
)

type ctRootOp string

const (
	Add    ctRootOp = "ADD"
	Remove ctRootOp = "REMOVE"
)

func main() {
	flag.Parse()
	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		panic("env variable NAMESPACE must be set")
	}

	ctx := signals.NewContext()

	var op ctRootOp
	switch ctRootOperation := *operation; ctRootOperation {
	case "add":
		op = Add
	case "remove":
		op = Remove
	default:
		logging.FromContext(ctx).Fatalf("No operation given, use --operation with [add,remove]")
	}

	versionInfo := version.GetVersionInfo()
	logging.FromContext(ctx).Infof("running managectroots Version: %s GitCommit: %s BuildDate: %s", versionInfo.GitVersion, versionInfo.GitCommit, versionInfo.BuildDate)

	logging.FromContext(ctx).Infof("%sing %s", op, *fulcioURL)

	config, err := rest.InClusterConfig()
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get InClusterConfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get clientset: %v", err)
	}

	// Fetch the fulcio Root CA
	u, err := url.Parse(*fulcioURL)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Invalid fulcioURL %s : %v", *fulcioURL, err)
	}
	client := fulcioclient.NewClient(u)
	root, err := client.RootCert()
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to fetch fulcio Root cert: %w", err)
	}

	logging.FromContext(ctx).Infof("Fulcio Root is: %+v", root)
	current := map[string][]byte{}
	nsSecret := clientset.CoreV1().Secrets(ns)
	secrets, err := nsSecret.Get(ctx, *secretName, metav1.GetOptions{})
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to get secret: %s/%s : %v", ns, *secretName, err)
	}
	current["private"] = secrets.Data["private"]
	current["public"] = secrets.Data["public"]
	current["rootca"] = secrets.Data["rootca"]
	for k, v := range secrets.Data {
		if strings.HasPrefix(k, "fulcio-") {
			current[k] = v
		}
	}
	// If the config is stored in the secret, we don't need to deal with the
	// configmap.
	var cm *corev1.ConfigMap
	if !*configInSecret {
		var err error
		cm, err = clientset.CoreV1().ConfigMaps(ns).Get(ctx, *cmname, metav1.GetOptions{})
		if err != nil {
			logging.FromContext(ctx).Panicf("Failed to get the configmap %s/%s: %v", ns, *cmname, err)
		}
		if cm.BinaryData == nil || cm.BinaryData[configKey] == nil {
			logging.FromContext(ctx).Fatalf("Configmap does not hold existing configmap %s/%s: %v", ns, *cmname, err)
		}
		current[configKey] = cm.BinaryData[configKey]
	} else {
		current[configKey] = secrets.Data[configKey]
	}

	conf, err := ctlog.Unmarshal(ctx, current)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to unmarshal: %s", err)
	}
	if op == Add {
		if err = conf.AddFulcioRoot(ctx, root.ChainPEM); err != nil {
			logging.FromContext(ctx).Infof("Failed to add Fulcio root: %v", err)
		}
	} else {
		if err = conf.RemoveFulcioRoot(ctx, root.ChainPEM); err != nil {
			logging.FromContext(ctx).Infof("Failed to remove Fulcio root: %v", err)
		}
	}

	// Marshal it and update configuration
	newConfig, err := conf.MarshalConfig(ctx)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to marshal config: %v", err)
	}
	if !*configInSecret {
		cm.BinaryData[configKey] = newConfig[configKey]
		if _, err = clientset.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
			logging.FromContext(ctx).Fatalf("Failed to update configmap %s/%s: %v", ns, *cmname, err)
		}
	}

	// Update the secret with the information
	secrets.Data = newConfig
	if _, err := nsSecret.Update(ctx, secrets, metav1.UpdateOptions{}); err != nil {
		logging.FromContext(ctx).Fatalf("Failed to udpate secret %s/%s: %v", ns, *secretName, err)
	}
	logging.FromContext(ctx).Infof("Config updated")
}
