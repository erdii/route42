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

package route42plugin

import (
	"context"
	"fmt"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/file"
	"github.com/coredns/coredns/request"
	"github.com/go-logr/logr"
	"github.com/miekg/dns"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	route42v1alpha1 "github.com/thetechnick/route42/api/v1alpha1"
	"github.com/thetechnick/route42/coredns/controllers"
)

const pluginName = "route42"

var scheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = route42v1alpha1.AddToScheme(scheme)
}

type zones interface {
	RLock()
	RUnlock()

	Zones() []string
	Zone(string) (*file.Zone, bool)
}

type route42plugin struct {
	Namespace string
	Next      plugin.Handler

	log   logr.Logger
	zones zones
}

func newRoute42Plugin(namespace string) (*route42plugin, error) {
	route42 := &route42plugin{
		Namespace: namespace,
		log:       ctrl.Log.WithName("route42"),
	}

	return route42, nil
}

func (p *route42plugin) Name() string { return pluginName }

func (p *route42plugin) ServeDNS(
	ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	qname := state.Name()

	log := p.log.WithValues("qname", qname, "qtype", state.Type())
	log.Info("serving")

	p.zones.RLock()
	defer p.zones.RUnlock()

	// check if we are managing the zone for the request
	zones := p.zones.Zones()
	zoneName := plugin.Zones(zones).Matches(qname)
	if zoneName == "" {
		log.WithValues("zones", zones).Info("zone not managed")
		return plugin.NextOrFailure(p.Name(), p.Next, ctx, w, r)
	}

	// get the zone object
	zone, ok := p.zones.Zone(zoneName)
	if !ok {
		return dns.RcodeServerFailure, nil
	}

	m := &dns.Msg{}
	m.SetReply(r)
	m.Authoritative = true
	var result file.Result
	m.Answer, m.Ns, m.Extra, result = zone.Lookup(ctx, state, qname)

	switch result {
	case file.Success:
	case file.NoData:
	case file.NameError:
		m.Rcode = dns.RcodeNameError
	case file.Delegation:
		m.Authoritative = false
	case file.ServerFailure:
		return dns.RcodeServerFailure, nil
	}

	return dns.RcodeSuccess, w.WriteMsg(m)
}

func (p *route42plugin) Run() error {
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("creating config: %w", err)
	}
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
		LeaderElection:     false,
		Namespace:          p.Namespace,
		Port:               0,
	})
	if err != nil {
		return fmt.Errorf("creating manager: %w", err)
	}

	// controllers
	zoneReconciler := controllers.NewZoneReconciler(
		mgr.GetClient(),
		ctrl.Log.WithName("controllers").WithName("Zone"),
	)
	p.zones = zoneReconciler
	if err = zoneReconciler.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("creating Zone controller: %w", err)
	}

	var stop chan struct{}
	go func() {
		if err := mgr.Start(stop); err != nil {
			panic(err)
		}
	}()

	return nil
}
