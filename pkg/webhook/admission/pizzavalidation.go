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

package admission

import (
	"fmt"
	"io/ioutil"
	"net/http"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"

	"github.com/programming-kubernetes/pizza-crd/pkg/apis/restaurant/install"
	"github.com/programming-kubernetes/pizza-crd/pkg/apis/restaurant/v1alpha1"
	"github.com/programming-kubernetes/pizza-crd/pkg/apis/restaurant/v1beta1"
	restaurantinformers "github.com/programming-kubernetes/pizza-crd/pkg/generated/informers/externalversions"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

func init() {
	utilruntime.Must(admissionv1beta1.AddToScheme(scheme))
	install.Install(scheme)
}

func ServePizzaValidation(informers restaurantinformers.SharedInformerFactory) func(http.ResponseWriter, *http.Request) {
	toppingInformer := informers.Restaurant().V1alpha1().Toppings().Informer()
	toppingLister := informers.Restaurant().V1alpha1().Toppings().Lister()

	return func(w http.ResponseWriter, req *http.Request) {
		if !toppingInformer.HasSynced() {
			responsewriters.InternalError(w, req, fmt.Errorf("informers not ready"))
			return
		}

		// read body
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			responsewriters.InternalError(w, req, fmt.Errorf("failed to read body: %v", err))
			return
		}

		// decode body as admission review
		reviewGVK := admissionv1beta1.SchemeGroupVersion.WithKind("AdmissionReview")
		obj, gvk, err := codecs.UniversalDeserializer().Decode(body, &reviewGVK, &admissionv1beta1.AdmissionReview{})
		if err != nil {
			responsewriters.InternalError(w, req, fmt.Errorf("failed to decode body: %v", err))
			return
		}
		review, ok := obj.(*admissionv1beta1.AdmissionReview)
		if !ok {
			responsewriters.InternalError(w, req, fmt.Errorf("unexpected GroupVersionKind: %s", gvk))
			return
		}
		if review.Request == nil {
			responsewriters.InternalError(w, req, fmt.Errorf("unexpected nil request"))
			return
		}
		review.Response = &admissionv1beta1.AdmissionResponse{
			UID: review.Request.UID,
		}

		// decode object
		if review.Request.Object.Object == nil {
			var err error
			review.Request.Object.Object, _, err = codecs.UniversalDeserializer().Decode(review.Request.Object.Raw, nil, nil)
			if err != nil {
				review.Response.Result = &metav1.Status{
					Message: err.Error(),
					Status:  metav1.StatusFailure,
				}
				responsewriters.WriteObject(http.StatusOK, gvk.GroupVersion(), codecs, review, w, req)
				return
			}
		}

		switch pizza := review.Request.Object.Object.(type) {
		case *v1alpha1.Pizza:
			for _, topping := range pizza.Spec.Toppings {
				if _, err := toppingLister.Get(topping); err != nil && !errors.IsNotFound(err) {
					responsewriters.InternalError(w, req, fmt.Errorf("failed to lookup topping %q: %v", topping, err))
					return
				} else if errors.IsNotFound(err) {
					review.Response.Result = &metav1.Status{
						Message: fmt.Sprintf("topping %q not known", topping),
						Status:  metav1.StatusFailure,
					}
					responsewriters.WriteObject(http.StatusOK, gvk.GroupVersion(), codecs, review, w, req)
					return
				}
			}
			review.Response.Allowed = true
		case *v1beta1.Pizza:
			for _, topping := range pizza.Spec.Toppings {
				if _, err := toppingLister.Get(topping.Name); err != nil && !errors.IsNotFound(err) {
					responsewriters.InternalError(w, req, fmt.Errorf("failed to lookup topping %q: %v", topping, err))
					return
				} else if errors.IsNotFound(err) {
					review.Response.Result = &metav1.Status{
						Message: fmt.Sprintf("topping %q not known", topping),
						Status:  metav1.StatusFailure,
					}
					responsewriters.WriteObject(http.StatusOK, gvk.GroupVersion(), codecs, review, w, req)
					return
				}
			}
			review.Response.Allowed = true
		default:
			review.Response.Result = &metav1.Status{
				Message: fmt.Sprintf("unexpected type %T", review.Request.Object.Object),
				Status:  metav1.StatusFailure,
			}
		}
		responsewriters.WriteObject(http.StatusOK, gvk.GroupVersion(), codecs, review, w, req)
	}
}
