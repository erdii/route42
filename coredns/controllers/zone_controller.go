/*
Copyright 2019 The MCP Authors.

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

package controllers

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/coredns/coredns/plugin/file"
	"github.com/go-logr/logr"
	"github.com/miekg/dns"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	route42v1alpha1 "github.com/thetechnick/route42/api/v1alpha1"
)

// ZoneReconciler reconciles a Zone object
type ZoneReconciler struct {
	client client.Client
	log    logr.Logger

	zones     map[string]*file.Zone
	zoneNames []string
	sync.RWMutex
}

func NewZoneReconciler(c client.Client, log logr.Logger) *ZoneReconciler {
	return &ZoneReconciler{
		client: c,
		log:    log,

		zones: map[string]*file.Zone{},
	}
}

func (r *ZoneReconciler) Zones() []string {
	return r.zoneNames
}

func (r *ZoneReconciler) Zone(zone string) (*file.Zone, bool) {
	z, ok := r.zones[zone]
	return z, ok
}

// +kubebuilder:rbac:groups=route42.thetechnick.ninja,resources=zones,verbs=get;list;watch
// +kubebuilder:rbac:groups=route42.thetechnick.ninja,resources=recordsets,verbs=get;list;watch

func (r *ZoneReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
	r.Lock()
	defer r.Unlock()

	log := r.log.WithValues("request", req.NamespacedName)

	ctx := context.Background()
	zones, err := r.listZones(ctx)
	if err != nil {
		return
	}

	var zoneNames []string
	zonesMap := map[string]*file.Zone{}
	for _, zone := range zones {
		zoneName := zone.Name + "."
		z := file.NewZone(zoneName, "")
		zonesMap[zoneName] = z
		zoneNames = append(zoneNames, zoneName)

		// SOA record
		rr, err := soaRecord(zoneName, &zone)
		if err != nil {
			return result, err
		}
		log.WithValues("rr", rr.String()).V(1).Info("add entry")
		_ = z.Insert(rr)

		// rest
		records, err := r.listRecords(ctx, zone.Name)
		if err != nil {
			return result, err
		}

		for _, record := range records {
			values := record.Values()
			if len(values) == 0 {
				continue
			}

			for _, v := range values {
				rfc1035 := fmt.Sprintf(
					"%s %d IN %s %s", record.DNSName, ttl(record.TTL), string(record.GetType()), v)
				rr, err := dns.NewRR(rfc1035)
				if err != nil {
					return result, fmt.Errorf("failed to create DNS record: %w", err)
				}
				log.WithValues("rr", rr.String()).V(1).Info("add entry")
				_ = z.Insert(rr)
			}
		}
	}

	r.zoneNames = zoneNames
	r.zones = zonesMap
	return
}

func (r *ZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(&source.Kind{Type: &route42v1alpha1.Zone{}}, &handler.EnqueueRequestForObject{}).
		Watches(&source.Kind{Type: &route42v1alpha1.RecordSet{}}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}

func (r *ZoneReconciler) listZones(ctx context.Context) ([]route42v1alpha1.Zone, error) {
	zoneList := &route42v1alpha1.ZoneList{}
	if err := r.client.List(ctx, zoneList); err != nil {
		return nil, err
	}
	return zoneList.Items, nil
}

func (r *ZoneReconciler) listRecords(ctx context.Context, zone string) (
	[]route42v1alpha1.Record, error) {
	recordSetList := &route42v1alpha1.RecordSetList{}
	if err := r.client.List(ctx, recordSetList); err != nil {
		return nil, err
	}

	var records []route42v1alpha1.Record
	for _, recordSet := range recordSetList.Items {
		if !strings.HasSuffix(recordSet.Record.DNSName, "."+zone) {
			continue
		}
		records = append(records, recordSet.Record)
	}

	return records, nil
}

func ttl(d metav1.Duration) int {
	return int(d.Duration.Seconds())
}

// Creates a SOA record for the given zone.
func soaRecord(zoneName string, zone *route42v1alpha1.Zone) (dns.RR, error) {
	soa := zone.Zone.SOA
	v := fmt.Sprintf("%s %s %d %d %d %d %d", soa.Master, soa.Admin, soa.Serial,
		ttl(soa.Refresh), ttl(soa.Retry), ttl(soa.Expire), ttl(soa.NegativeTTL))
	rfc1035 := fmt.Sprintf("%s %d IN %s %s", zoneName, ttl(soa.TTL), "SOA", v)
	return dns.NewRR(rfc1035)
}
