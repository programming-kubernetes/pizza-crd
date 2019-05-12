/*
Copyright 2019 The Kubernetes Authors.

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

package conversion

import (
	"fmt"
	"io/ioutil"
	"net/http"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"

	"github.com/programming-kubernetes/pizza-crd/pkg/apis/restaurant/install"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

func init() {
	utilruntime.Must(apiextensionsv1beta1.AddToScheme(scheme))
	install.Install(scheme)
}

func Serve(w http.ResponseWriter, req *http.Request) {
	// read body
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		responsewriters.InternalError(w, req, fmt.Errorf("failed to read body: %v", err))
		return
	}

	// decode body as conversion review
	reviewGVK := apiextensionsv1beta1.SchemeGroupVersion.WithKind("ConversionReview")
	obj, gvk, err := codecs.UniversalDeserializer().Decode(body, &reviewGVK, &apiextensionsv1beta1.ConversionReview{})
	if err != nil {
		responsewriters.InternalError(w, req, fmt.Errorf("failed to decode body: %v", err))
		return
	}
	review, ok := obj.(*apiextensionsv1beta1.ConversionReview)
	if !ok {
		responsewriters.InternalError(w, req, fmt.Errorf("unexpected GroupVersionKind: %s", gvk))
		return
	}
	if review.Request == nil {
		responsewriters.InternalError(w, req, fmt.Errorf("unexpected nil request"))
		return
	}

	// convert objects
	review.Response = &apiextensionsv1beta1.ConversionResponse{
		UID: review.Request.UID,
		Result: metav1.Status{
			Status: metav1.StatusSuccess,
		},
	}
	var objs []runtime.Object
	for _, in := range review.Request.Objects {
		if in.Object == nil {
			var err error
			in.Object, _, err = codecs.UniversalDeserializer().Decode(in.Raw, nil, nil)
			if err != nil {
				review.Response.Result = metav1.Status{
					Message: err.Error(),
					Status:  metav1.StatusFailure,
				}
				break
			}
		}

		obj, err := convert(in.Object, review.Request.DesiredAPIVersion)
		if err != nil {
			review.Response.Result = metav1.Status{
				Message: err.Error(),
				Status:  metav1.StatusFailure,
			}
			break
		}
		objs = append(objs, obj)
	}
	if review.Response.Result.Status == metav1.StatusSuccess {
		for _, obj = range objs {
			review.Response.ConvertedObjects = append(review.Response.ConvertedObjects, runtime.RawExtension{
				Object: obj,
			})
		}
	}

	// write negotiated response
	responsewriters.WriteObject(http.StatusOK, gvk.GroupVersion(), codecs, review, w, req)
}
