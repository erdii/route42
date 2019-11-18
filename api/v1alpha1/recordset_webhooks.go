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
	"net"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// +kubebuilder:webhook:path=/mutate-route42-thetechnick-ninja-v1alpha1-recordset,mutating=true,failurePolicy=fail,groups=route42.thetechnick.ninja,resources=recordsets,verbs=create;update,versions=v1alpha1,name=mutation-recordset.route42.thetechnick.ninja

// +kubebuilder:webhook:verbs=create;update,path=/validate-route42-thetechnick-ninja-v1alpha1-recordset,mutating=false,failurePolicy=fail,groups=route42.thetechnick.ninja,resources=recordsets,versions=v1alpha1,name=validation-recordset.route42.thetechnick.ninja

var (
	recordSetlog                   = logf.Log.WithName("recordSet-resource")
	_            webhook.Defaulter = (*Zone)(nil)
	_            webhook.Validator = (*Zone)(nil)
)

func (r *RecordSet) Default() {
	recordSetlog.Info("default", "Zone",
		types.NamespacedName{Name: r.Name, Namespace: r.Namespace})

	r.Record.Type = r.Record.GetType()
}

func (r *RecordSet) ValidateCreate() error {
	recordSetlog.Info("validate create", "Zone",
		types.NamespacedName{Name: r.Name, Namespace: r.Namespace})
	return r.validate(nil)
}

func (r *RecordSet) ValidateUpdate(old runtime.Object) error {
	recordSetlog.Info("validate update", "Zone",
		types.NamespacedName{Name: r.Name, Namespace: r.Namespace})
	return r.validate(old.(*RecordSet))
}

func (r *RecordSet) ValidateDelete() error {
	recordSetlog.Info("validate delete", "Zone",
		types.NamespacedName{Name: r.Name, Namespace: r.Namespace})
	return nil
}

func (r *RecordSet) validate(old *RecordSet) error {
	var allErrs field.ErrorList

	if err := validateName(
		field.NewPath("record").Child("dnsName"), r.Record.DNSName); err != nil {
		allErrs = append(allErrs, err)
	}

	if r.Record.Type == RecordTypeUnknown || r.Record.Type == "" {
		path := field.NewPath("record").Child("type")
		err := field.Invalid(path, r.Record.Type, "unknown record type")
		allErrs = append(allErrs, err)
	}

	switch r.Record.Type {
	case RecordTypeA:
		allErrs = append(allErrs, filterNil(
			validateA(r.Record.A),
			noAAAA(r.Record.AAAA),
			noTXT(r.Record.TXT),
			noCName(r.Record.CName),
			noNS(r.Record.NS),
			noMX(r.Record.MX),
			noSRV(r.Record.SRV),
		)...)

	case RecordTypeAAAA:
		allErrs = append(allErrs, filterNil(
			validateAAAA(r.Record.AAAA),
			noA(r.Record.A),
			noTXT(r.Record.TXT),
			noCName(r.Record.CName),
			noNS(r.Record.NS),
			noMX(r.Record.MX),
			noSRV(r.Record.SRV),
		)...)

	case RecordTypeTXT:
		allErrs = append(allErrs, filterNil(
			nil,
			noA(r.Record.A),
			noAAAA(r.Record.AAAA),
			noCName(r.Record.CName),
			noNS(r.Record.NS),
			noMX(r.Record.MX),
			noSRV(r.Record.SRV),
		)...)

	case RecordTypeCName:
		allErrs = append(allErrs, filterNil(
			nil,
			noA(r.Record.A),
			noAAAA(r.Record.AAAA),
			noTXT(r.Record.TXT),
			noNS(r.Record.NS),
			noMX(r.Record.MX),
			noSRV(r.Record.SRV),
		)...)

	case RecordTypeNS:
		allErrs = append(allErrs, filterNil(
			nil,
			noA(r.Record.A),
			noAAAA(r.Record.AAAA),
			noTXT(r.Record.TXT),
			noCName(r.Record.CName),
			noMX(r.Record.MX),
			noSRV(r.Record.SRV),
		)...)

	case RecordTypeMX:
		allErrs = append(allErrs, filterNil(
			nil,
			noA(r.Record.A),
			noAAAA(r.Record.AAAA),
			noTXT(r.Record.TXT),
			noCName(r.Record.CName),
			noNS(r.Record.NS),
			noSRV(r.Record.SRV),
		)...)

	case RecordTypeSRV:
		allErrs = append(allErrs, filterNil(
			nil,
			noA(r.Record.A),
			noAAAA(r.Record.AAAA),
			noTXT(r.Record.TXT),
			noCName(r.Record.CName),
			noNS(r.Record.NS),
			noMX(r.Record.MX),
		)...)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: GroupVersion.Group, Kind: "Zone"},
		r.Name, allErrs)
}

func filterNil(fields []*field.Error, errs ...*field.Error) []*field.Error {
	for _, err := range errs {
		if err == nil {
			continue
		}
		fields = append(fields, err)
	}
	return fields
}

func noA(a []string) *field.Error {
	if len(a) == 0 {
		return nil
	}
	path := field.NewPath("record").Child("a")
	return field.Invalid(path, a, "can not contain multiple types of records")
}

func validateA(a []string) []*field.Error {
	var errs []*field.Error
	for i, entry := range a {
		path := field.NewPath("record").Child("a").Index(i)

		ip := net.ParseIP(entry)
		if ip == nil || ip.To4() == nil {
			errs = append(errs, field.Invalid(path, entry, "not a valid IPv4 address"))
		}
	}
	return errs
}

func noAAAA(aaaa []string) *field.Error {
	if len(aaaa) == 0 {
		return nil
	}
	path := field.NewPath("record").Child("aaaa")
	return field.Invalid(path, aaaa, "can not contain multiple types of records")
}

func validateAAAA(a []string) []*field.Error {
	var errs []*field.Error
	for i, entry := range a {
		path := field.NewPath("record").Child("aaaa").Index(i)

		ip := net.ParseIP(entry)
		if ip == nil || ip.To16() == nil {
			errs = append(errs, field.Invalid(path, entry, "not a valid IPv6 address"))
		}
	}
	return errs
}

func noTXT(txt []string) *field.Error {
	if len(txt) == 0 {
		return nil
	}
	path := field.NewPath("record").Child("txt")
	return field.Invalid(path, txt, "can not contain multiple types of records")
}

func noCName(cname *string) *field.Error {
	if cname == nil {
		return nil
	}
	path := field.NewPath("record").Child("cname")
	return field.Invalid(path, cname, "can not contain multiple types of records")
}

func noNS(ns []string) *field.Error {
	if len(ns) == 0 {
		return nil
	}
	path := field.NewPath("record").Child("ns")
	return field.Invalid(path, ns, "can not contain multiple types of records")
}

func noMX(mx []MX) *field.Error {
	if len(mx) == 0 {
		return nil
	}
	path := field.NewPath("record").Child("mx")
	return field.Invalid(path, mx, "can not contain multiple types of records")
}

func noSRV(srv []SRV) *field.Error {
	if len(srv) == 0 {
		return nil
	}
	path := field.NewPath("record").Child("srv")
	return field.Invalid(path, srv, "can not contain multiple types of records")
}

func (r *RecordSet) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}
