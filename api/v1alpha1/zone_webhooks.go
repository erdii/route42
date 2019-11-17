/*
Copyright 2019 The Route42 Authors.

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

package v1alpha1

import (
	"time"

	"github.com/miekg/dns"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// +kubebuilder:webhook:path=/mutate-route42-thetechnick-ninja-v1alpha1-zone,mutating=true,failurePolicy=fail,groups=route42.thetechnick.ninja,resources=zones,verbs=create;update,versions=v1alpha1,name=mutation-zone.route42.thetechnick.ninja

// +kubebuilder:webhook:verbs=create;update,path=/validate-route42-thetechnick-ninja-v1alpha1-zone,mutating=false,failurePolicy=fail,groups=route42.thetechnick.ninja,resources=zones,versions=v1alpha1,name=validation-zone.route42.thetechnick.ninja

var (
	zonelog                   = logf.Log.WithName("zone-resource")
	_       webhook.Defaulter = (*Zone)(nil)
	_       webhook.Validator = (*Zone)(nil)
)

func (z *Zone) Default() {
	zonelog.Info("default", "Zone",
		types.NamespacedName{Name: z.Name, Namespace: z.Namespace})

	if z.Zone.SOA.Refresh.Duration == 0 {
		z.Zone.SOA.Refresh.Duration = time.Hour * 24
	}
	if z.Zone.SOA.Retry.Duration == 0 {
		z.Zone.SOA.Retry.Duration = time.Hour * 2
	}
	if z.Zone.SOA.Expire.Duration == 0 {
		z.Zone.SOA.Expire.Duration = time.Hour * 1000
	}
	if z.Zone.SOA.NegativeTTL.Duration == 0 {
		z.Zone.SOA.NegativeTTL.Duration = time.Hour * 24 * 2
	}
}

func (z *Zone) ValidateCreate() error {
	zonelog.Info("validate create", "Zone",
		types.NamespacedName{Name: z.Name, Namespace: z.Namespace})
	return z.validate(nil)
}

func (z *Zone) ValidateUpdate(old runtime.Object) error {
	zonelog.Info("validate update", "Zone",
		types.NamespacedName{Name: z.Name, Namespace: z.Namespace})
	return z.validate(old.(*Zone))
}

func (z *Zone) ValidateDelete() error {
	zonelog.Info("validate delete", "Zone",
		types.NamespacedName{Name: z.Name, Namespace: z.Namespace})
	return nil
}

func (z *Zone) validate(old *Zone) error {
	var allErrs field.ErrorList

	if err := validateName(
		field.NewPath("metadata").Child("name"), z.Name); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateName(
		field.NewPath("zone").Child("soa").Child("master"), z.Zone.SOA.Master); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateName(
		field.NewPath("zone").Child("soa").Child("admin"), z.Zone.SOA.Admin); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: GroupVersion.Group, Kind: "Zone"},
		z.Name, allErrs)
}

func (z *Zone) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(z).
		Complete()
}

func validateName(path *field.Path, host string) *field.Error {
	_, ok := dns.IsDomainName(host)
	if !ok {
		return field.Invalid(path, host, "not a valid domain")
	}
	return nil
}
