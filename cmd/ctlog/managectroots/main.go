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

	fulcioclient "github.com/sigstore/fulcio/pkg/api"
	"github.com/sigstore/scaffolding/pkg/ctlog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
	"sigs.k8s.io/release-utils/version"
)

var (
	secretName = flag.String("secret", "ctlog-secrets", "Name of the secret to fetch private key for CTLog.")
	fulcioURL  = flag.String("fulcio-url", "http://fulcio.fulcio-system.svc", "Where to fetch the fulcio Root CA from.")
	operation  = flag.String("operation", "", "Operation to perform for the specified fulcio [add,remove]")
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
	current["fulcio"] = secrets.Data["fulcio"]

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
	newConfig, err := conf.MarshalConfig()
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to marshal config: %v", err)
	}

	// Update the secret with the information
	secrets.Data = newConfig
	if _, err := nsSecret.Update(ctx, secrets, metav1.UpdateOptions{}); err != nil {
		logging.FromContext(ctx).Fatalf("Failed to udpate secret %s/%s: %v", ns, *secretName, err)
	}
	logging.FromContext(ctx).Infof("Config updated")
}
