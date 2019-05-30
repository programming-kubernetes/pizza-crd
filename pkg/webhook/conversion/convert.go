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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"

	"github.com/programming-kubernetes/pizza-crd/pkg/apis/restaurant/v1alpha1"
	"github.com/programming-kubernetes/pizza-crd/pkg/apis/restaurant/v1beta1"
)

func convert(in runtime.Object, apiVersion string) (runtime.Object, error) {
	switch in := in.(type) {
	case *v1alpha1.Pizza:
		if apiVersion != v1beta1.SchemeGroupVersion.String() {
			return nil, fmt.Errorf("cannot convert %s to %s", v1alpha1.SchemeGroupVersion, apiVersion)
		}
		klog.V(2).Infof("Converting %s/%s from %s to %s", in.Namespace, in.Name, v1alpha1.SchemeGroupVersion, apiVersion)

		out := &v1beta1.Pizza{
			TypeMeta:   in.TypeMeta,
			ObjectMeta: in.ObjectMeta,
			Status: v1beta1.PizzaStatus{
				Cost: in.Status.Cost,
			},
		}
		out.TypeMeta.APIVersion = apiVersion

		idx := map[string]int{}
		for _, top := range in.Spec.Toppings {
			if i, duplicate := idx[top]; duplicate {
				out.Spec.Toppings[i].Quantity++
				continue
			}
			idx[top] = len(out.Spec.Toppings)
			out.Spec.Toppings = append(out.Spec.Toppings, v1beta1.PizzaTopping{
				Name:     top,
				Quantity: 1,
			})
		}

		return out, nil

	case *v1beta1.Pizza:
		if apiVersion != v1alpha1.SchemeGroupVersion.String() {
			return nil, fmt.Errorf("cannot convert %s to %s", v1beta1.SchemeGroupVersion, apiVersion)
		}
		klog.V(2).Infof("Converting %s/%s from %s to %s", in.Namespace, in.Name, v1alpha1.SchemeGroupVersion, apiVersion)

		out := &v1alpha1.Pizza{
			TypeMeta:   in.TypeMeta,
			ObjectMeta: in.ObjectMeta,
			Status: v1alpha1.PizzaStatus{
				Cost: in.Status.Cost,
			},
		}
		out.TypeMeta.APIVersion = apiVersion

		for i := range in.Spec.Toppings {
			for j := 0; j < in.Spec.Toppings[i].Quantity; j++ {
				out.Spec.Toppings = append(out.Spec.Toppings, in.Spec.Toppings[i].Name)
			}
		}

		return out, nil

	default:
	}
	klog.V(2).Infof("Unknown type %T", in)
	return nil, fmt.Errorf("unknown type %T", in)
}
