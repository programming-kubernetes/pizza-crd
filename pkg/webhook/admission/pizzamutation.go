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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/appscode/jsonpatch"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/klog"

	"github.com/programming-kubernetes/pizza-crd/pkg/apis/restaurant/v1alpha1"
	"github.com/programming-kubernetes/pizza-crd/pkg/apis/restaurant/v1beta1"
)

func ServePizzaAdmit(w http.ResponseWriter, req *http.Request) {
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

	orig := review.Request.Object.Raw
	var bs []byte
	switch pizza := review.Request.Object.Object.(type) {
	case *v1alpha1.Pizza:
		// default toppings
		if len(pizza.Spec.Toppings) == 0 {
			pizza.Spec.Toppings = []string{"tomato", "mozzarella", "salami"}
		}
		bs, err = json.Marshal(pizza)
		if err != nil {
			responsewriters.InternalError(w, req, fmt.Errorf("unexpected encoding error: %v", err))
			return
		}

	case *v1beta1.Pizza:
		// default toppings
		if len(pizza.Spec.Toppings) == 0 {
			pizza.Spec.Toppings = []v1beta1.PizzaTopping{
				{"tomato", 1},
				{"mozzarella", 1},
				{"salami", 1},
			}
		}
		bs, err = json.Marshal(pizza)
		if err != nil {
			responsewriters.InternalError(w, req, fmt.Errorf("unexpected encoding error: %v", err))
			return
		}

	default:
		review.Response.Result = &metav1.Status{
			Message: fmt.Sprintf("unexpected type %T", review.Request.Object.Object),
			Status:  metav1.StatusFailure,
		}
		responsewriters.WriteObject(http.StatusOK, gvk.GroupVersion(), codecs, review, w, req)
		return
	}

	klog.V(2).Infof("Defaulting %s/%s in version %s", review.Request.Namespace, review.Request.Name, gvk)

	// compare original and defaulted version
	ops, err := jsonpatch.CreatePatch(orig, bs)
	if err != nil {
		responsewriters.InternalError(w, req, fmt.Errorf("unexpected diff error: %v", err))
		return
	}
	review.Response.Patch, err = json.Marshal(ops)
	if err != nil {
		responsewriters.InternalError(w, req, fmt.Errorf("unexpected patch encoding error: %v", err))
		return
	}
	typ := admissionv1beta1.PatchTypeJSONPatch
	review.Response.PatchType = &typ
	review.Response.Allowed = true

	responsewriters.WriteObject(http.StatusOK, gvk.GroupVersion(), codecs, review, w, req)
}
