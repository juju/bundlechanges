// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package bundlechanges_test

import (
	"encoding/json"
	"strings"
	"testing"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charm.v6-unstable"

	"github.com/juju/bundlechanges"
)

type changesSuite struct{}

var _ = gc.Suite(&changesSuite{})

func TestPackage(t *testing.T) {
	gc.TestingT(t)
}

var fromDataTests = []struct {
	// about describes the test.
	about string
	// content is the YAML encoded bundle content.
	content string
	// expected holds the expected changes required to deploy the bundle.
	expected []*bundlechanges.Change
}{{
	about: "minimal bundle",
	content: `
        services:
            django:
                charm: django
    `,
	expected: []*bundlechanges.Change{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Args:   []interface{}{"django"},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Args: []interface{}{
			"django",
			"django",
			map[string]interface{}{},
		},
		Requires: []string{"addCharm-0"},
	}},
}, {
	about: "simple bundle",
	content: `
        services:
            mediawiki:
                charm: cs:precise/mediawiki-10
                num_units: 1
                options:
                    debug: false
                annotations:
                    gui-x: "609"
                    gui-y: "-15"
            mysql:
                charm: cs:precise/mysql-28
                num_units: 1
        series: trusty
        relations:
            - - mediawiki:db
              - mysql:db
    `,
	expected: []*bundlechanges.Change{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Args:   []interface{}{"cs:precise/mediawiki-10"},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Args: []interface{}{
			"cs:precise/mediawiki-10",
			"mediawiki",
			map[string]interface{}{"debug": false},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "setAnnotations-2",
		Method: "setAnnotations",
		Args: []interface{}{
			"$deploy-1",
			"service",
			map[string]string{"gui-x": "609", "gui-y": "-15"},
		},
		Requires: []string{"deploy-1"},
	}, {
		Id:     "addCharm-3",
		Method: "addCharm",
		Args:   []interface{}{"cs:precise/mysql-28"},
	}, {
		Id:     "deploy-4",
		Method: "deploy",
		Args: []interface{}{
			"cs:precise/mysql-28",
			"mysql",
			map[string]interface{}{},
		},
		Requires: []string{"addCharm-3"},
	}},
}, {
	about: "bundle with machines and units placement",
	content: `
        services:
            django:
                charm: cs:trusty/django-42
                num_units: 2
                to:
                    - 1
                    - lxc:2
            haproxy:
                charm: cs:trusty/haproxy-47
                num_units: 2
                to:
                    - lxc:django/0
                    - new
                options:
                    bad: wolf
                    number: 42.47
        machines:
            1:
                series: trusty
            2:
    `,
	expected: []*bundlechanges.Change{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Args:   []interface{}{"cs:trusty/django-42"},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Args: []interface{}{
			"cs:trusty/django-42",
			"django",
			map[string]interface{}{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "addCharm-2",
		Method: "addCharm",
		Args:   []interface{}{"cs:trusty/haproxy-47"},
	}, {
		Id:     "deploy-3",
		Method: "deploy",
		Args: []interface{}{
			"cs:trusty/haproxy-47",
			"haproxy",
			map[string]interface{}{"bad": "wolf", "number": 42.47},
		},
		Requires: []string{"addCharm-2"},
	}, {
		Id:     "addMachines-4",
		Method: "addMachines",
		Args: []interface{}{
			map[string]string{"series": "trusty", "constraints": ""},
		},
	}, {
		Id:     "addMachines-5",
		Method: "addMachines",
		Args: []interface{}{
			map[string]string{"series": "", "constraints": ""},
		},
	}},
}}

func (s *changesSuite) TestFromData(c *gc.C) {
	for i, test := range fromDataTests {
		c.Logf("test %d: %s", i, test.about)

		// Retrieve and validate the bundle data.
		data, err := charm.ReadBundleData(strings.NewReader(test.content))
		c.Assert(err, jc.ErrorIsNil)
		err = data.Verify(nil)
		c.Assert(err, jc.ErrorIsNil)

		// Check that the changes are what we expect.
		changes := bundlechanges.FromData(data)
		b, err := json.MarshalIndent(changes, "", "  ")
		c.Assert(err, jc.ErrorIsNil)
		c.Logf("obtained changes: %s", b)
		c.Assert(changes, jc.DeepEquals, test.expected)
	}
}
