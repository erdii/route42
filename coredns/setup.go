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
	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func init() { plugin.Register(pluginName, setup) }

func setup(c *caddy.Controller) error {
	ctrl.SetLogger(zap.New(func(options *zap.Options) {
		options.Development = true
	}))

	for c.Next() {
		var namespace string
		if c.NextBlock() {
			switch c.Val() {
			case "namespace":
				if !c.NextArg() {
					return c.ArgErr()
				}
				namespace = c.Val()

			default:
				if c.Val() != "}" {
					return c.Errf("unknown property '%s'", c.Val())
				}
			}
		}

		r, err := newRoute42Plugin(namespace)
		if err != nil {
			return plugin.Error(pluginName, err)
		}

		go func() {
			if err := r.Run(); err != nil {
				panic(err)
			}
		}()
		dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
			r.Next = next
			return r
		})
	}
	return nil
}
