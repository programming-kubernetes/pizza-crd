/*
Copyright 2018 The Kubernetes Authors.

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

package main

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/handlers"
	"github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/options"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/cli/globalflag"

	"github.com/programming-kubernetes/pizza-crd/pkg/generated/clientset/versioned"
	restaurantinformers "github.com/programming-kubernetes/pizza-crd/pkg/generated/informers/externalversions"
	"github.com/programming-kubernetes/pizza-crd/pkg/webhook/admission"
	"github.com/programming-kubernetes/pizza-crd/pkg/webhook/conversion"
)

func NewDefaultOptions() *Options {
	o := &Options{
		*options.NewSecureServingOptions(),
	}
	o.SecureServing.ServerCert.PairName = "pizza-crd-webhook"
	return o
}

type Options struct {
	SecureServing options.SecureServingOptions
}

type Config struct {
	SecureServing *server.SecureServingInfo
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.SecureServing.AddFlags(fs)
}

func (o *Options) Config() (*Config, error) {
	if err := o.SecureServing.MaybeDefaultWithSelfSignedCerts("0.0.0.0", nil, nil); err != nil {
		return nil, err
	}

	c := &Config{}

	if err := o.SecureServing.ApplyTo(&c.SecureServing); err != nil {
		return nil, err
	}

	return c, nil
}

func main() {
	// parse flags
	opt := NewDefaultOptions()
	fs := pflag.NewFlagSet("pizza-crd-webhook", pflag.ExitOnError)
	globalflag.AddGlobalFlags(fs, "pizza-crd-webhook")
	opt.AddFlags(fs)
	if err := fs.Parse(os.Args); err != nil {
		panic(err)
	}

	// create runtime config
	cfg, err := opt.Config()
	if err != nil {
		panic(err)
	}

	// create client
	config, err := rest.InClusterConfig()
	if err != nil {
		// fallback to home directory
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		kubeconfig := filepath.Join(home, ".kube", "config")
		if envvar := os.Getenv("KUBECONFIG"); len(envvar) > 0 {
			kubeconfig = envvar
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err)
		}
	}
	clientset, err := versioned.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	stopCh := server.SetupSignalHandler()

	// register handlers
	restaurantInformers := restaurantinformers.NewSharedInformerFactory(clientset, time.Minute*5)
	mux := http.NewServeMux()
	mux.Handle("/convert/v1beta1/pizza", http.HandlerFunc(conversion.Serve))
	mux.Handle("/admit/v1beta1/pizza", http.HandlerFunc(admission.ServePizzaAdmit))
	mux.Handle("/validate/v1beta1/pizza", http.HandlerFunc(admission.ServePizzaValidation(restaurantInformers)))
	restaurantInformers.Start(stopCh)

	// run server
	if doneCh, err := cfg.SecureServing.Serve(handlers.LoggingHandler(os.Stdout, mux), time.Second*30, stopCh); err != nil {
		panic(err)
	} else {
		<-doneCh
	}
}
