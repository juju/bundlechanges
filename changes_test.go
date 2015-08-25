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
		Id:     "addService-1",
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
		Id:     "addService-1",
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
			"$addService-1",
			"service",
			map[string]string{"gui-x": "609", "gui-y": "-15"},
		},
		Requires: []string{"addService-1"},
	}, {
		Id:     "addCharm-3",
		Method: "addCharm",
		Args:   []interface{}{"cs:precise/mysql-28"},
	}, {
		Id:     "addService-4",
		Method: "deploy",
		Args: []interface{}{
			"cs:precise/mysql-28",
			"mysql",
			map[string]interface{}{},
		},
		Requires: []string{"addCharm-3"},
	}, {
		Id:       "addRelation-5",
		Method:   "addRelation",
		Args:     []interface{}{"$addService-1:db", "$addService-4:db"},
		Requires: []string{"addService-1", "addService-4"},
	}, {
		Id:       "addUnit-6",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, nil},
		Requires: []string{"addService-1"},
	}, {
		Id:       "addUnit-7",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-4", 1, nil},
		Requires: []string{"addService-4"},
	}},
}, {
	about: "machines and units placement",
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
		Id:     "addService-1",
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
		Id:     "addService-3",
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
	}, {
		Id:       "addUnit-6",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addMachines-4"},
		Requires: []string{"addService-1", "addMachines-4"},
	}, {
		Id:     "addMachines-10",
		Method: "addMachines",
		Args: []interface{}{map[string]string{
			"containerType": "lxc",
			"parentId":      "$addMachines-5",
		}},
		Requires: []string{"addMachines-5"},
	}, {
		Id:     "addMachines-11",
		Method: "addMachines",
		Args: []interface{}{map[string]string{
			"containerType": "lxc",
			"parentId":      "$addUnit-6",
		}},
		Requires: []string{"addUnit-6"},
	}, {
		Id:     "addMachines-12",
		Method: "addMachines",
	}, {
		Id:       "addUnit-7",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addMachines-10"},
		Requires: []string{"addService-1", "addMachines-10"},
	}, {
		Id:       "addUnit-8",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-3", 1, "$addMachines-11"},
		Requires: []string{"addService-3", "addMachines-11"},
	}, {
		Id:       "addUnit-9",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-3", 1, "$addMachines-12"},
		Requires: []string{"addService-3", "addMachines-12"},
	}},
}, {
	about: "machines with constraints and annotations",
	content: `
        services:
            django:
                charm: cs:trusty/django-42
                num_units: 2
                to:
                    - 1
                    - new
        machines:
            1:
                constraints: "cpu-cores=4"
                annotations:
                    foo: bar
    `,
	expected: []*bundlechanges.Change{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Args:   []interface{}{"cs:trusty/django-42"},
	}, {
		Id:     "addService-1",
		Method: "deploy",
		Args: []interface{}{
			"cs:trusty/django-42",
			"django",
			map[string]interface{}{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "addMachines-2",
		Method: "addMachines",
		Args: []interface{}{
			map[string]string{"series": "", "constraints": "cpu-cores=4"},
		},
	}, {
		Id:     "setAnnotations-3",
		Method: "setAnnotations",
		Args: []interface{}{
			"$addMachines-2",
			"machine",
			map[string]string{"foo": "bar"},
		},
		Requires: []string{"addMachines-2"},
	}, {
		Id:       "addUnit-4",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addMachines-2"},
		Requires: []string{"addService-1", "addMachines-2"},
	}, {
		Id:     "addMachines-6",
		Method: "addMachines",
	}, {
		Id:       "addUnit-5",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addMachines-6"},
		Requires: []string{"addService-1", "addMachines-6"},
	}},
}, {
	about: "endpoint without relation name",
	content: `
        services:
            mediawiki:
                charm: cs:precise/mediawiki-10
            mysql:
                charm: cs:precise/mysql-28
        relations:
            - - mediawiki:db
              - mysql
    `,
	expected: []*bundlechanges.Change{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Args:   []interface{}{"cs:precise/mediawiki-10"},
	}, {
		Id:     "addService-1",
		Method: "deploy",
		Args: []interface{}{
			"cs:precise/mediawiki-10",
			"mediawiki",
			map[string]interface{}{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "addCharm-2",
		Method: "addCharm",
		Args:   []interface{}{"cs:precise/mysql-28"},
	}, {
		Id:     "addService-3",
		Method: "deploy",
		Args: []interface{}{
			"cs:precise/mysql-28",
			"mysql",
			map[string]interface{}{},
		},
		Requires: []string{"addCharm-2"},
	}, {
		Id:       "addRelation-4",
		Method:   "addRelation",
		Args:     []interface{}{"$addService-1:db", "$addService-3"},
		Requires: []string{"addService-1", "addService-3"},
	}},
}, {
	about: "unit placed in service",
	content: `
        services:
            wordpress:
                charm: wordpress
                num_units: 3
            django:
                charm: cs:trusty/django-42
                num_units: 2
                to: [wordpress]
    `,
	expected: []*bundlechanges.Change{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Args:   []interface{}{"cs:trusty/django-42"},
	}, {
		Id:     "addService-1",
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
		Args:   []interface{}{"wordpress"},
	}, {
		Id:     "addService-3",
		Method: "deploy",
		Args: []interface{}{
			"wordpress",
			"wordpress",
			map[string]interface{}{},
		},
		Requires: []string{"addCharm-2"},
	}, {
		Id:       "addUnit-6",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-3", 1, nil},
		Requires: []string{"addService-3"},
	}, {
		Id:       "addUnit-7",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-3", 1, nil},
		Requires: []string{"addService-3"},
	}, {
		Id:       "addUnit-8",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-3", 1, nil},
		Requires: []string{"addService-3"},
	}, {
		Id:       "addUnit-4",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addUnit-6"},
		Requires: []string{"addService-1", "addUnit-6"},
	}, {
		Id:       "addUnit-5",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addUnit-7"},
		Requires: []string{"addService-1", "addUnit-7"},
	}},
}, {
	about: "unit co-location with other units",
	content: `
        services:
            memcached:
                charm: cs:trusty/mem-47
                num_units: 3
                to: [1, new]
            django:
                charm: cs:trusty/django-42
                num_units: 5
                to:
                    - memcached/0
                    - lxc:memcached/1
                    - lxc:memcached/2
                    - kvm:ror
            ror:
                charm: vivid/rails
                num_units: 2
                to:
                    - new
                    - 1
        machines:
            1:
                series: trusty
    `,
	expected: []*bundlechanges.Change{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Args:   []interface{}{"cs:trusty/django-42"},
	}, {
		Id:     "addService-1",
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
		Args:   []interface{}{"cs:trusty/mem-47"},
	}, {
		Id:     "addService-3",
		Method: "deploy",
		Args: []interface{}{
			"cs:trusty/mem-47",
			"memcached",
			map[string]interface{}{},
		},
		Requires: []string{"addCharm-2"},
	}, {
		Id:     "addCharm-4",
		Method: "addCharm",
		Args:   []interface{}{"vivid/rails"},
	}, {
		Id:     "addService-5",
		Method: "deploy",
		Args: []interface{}{
			"vivid/rails",
			"ror",
			map[string]interface{}{},
		},
		Requires: []string{"addCharm-4"},
	}, {
		Id:     "addMachines-6",
		Method: "addMachines",
		Args: []interface{}{map[string]string{
			"series":      "trusty",
			"constraints": "",
		}},
	}, {
		Id:       "addUnit-12",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-3", 1, "$addMachines-6"},
		Requires: []string{"addService-3", "addMachines-6"},
	}, {
		Id:       "addUnit-16",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-5", 1, "$addMachines-6"},
		Requires: []string{"addService-5", "addMachines-6"},
	}, {
		Id:     "addMachines-20",
		Method: "addMachines",
		Args: []interface{}{map[string]string{
			"containerType": "kvm",
			"parentId":      "$addUnit-16",
		}},
		Requires: []string{"addUnit-16"},
	}, {
		Id:     "addMachines-21",
		Method: "addMachines",
	}, {
		Id:     "addMachines-22",
		Method: "addMachines",
	}, {
		Id:     "addMachines-23",
		Method: "addMachines",
	}, {
		Id:       "addUnit-7",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addUnit-12"},
		Requires: []string{"addService-1", "addUnit-12"},
	}, {
		Id:       "addUnit-11",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addMachines-20"},
		Requires: []string{"addService-1", "addMachines-20"},
	}, {
		Id:       "addUnit-13",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-3", 1, "$addMachines-21"},
		Requires: []string{"addService-3", "addMachines-21"},
	}, {
		Id:       "addUnit-14",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-3", 1, "$addMachines-22"},
		Requires: []string{"addService-3", "addMachines-22"},
	}, {
		Id:       "addUnit-15",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-5", 1, "$addMachines-23"},
		Requires: []string{"addService-5", "addMachines-23"},
	}, {
		Id:     "addMachines-17",
		Method: "addMachines",
		Args: []interface{}{map[string]string{
			"containerType": "lxc",
			"parentId":      "$addUnit-13",
		}},
		Requires: []string{"addUnit-13"},
	}, {
		Id:     "addMachines-18",
		Method: "addMachines",
		Args: []interface{}{map[string]string{
			"containerType": "lxc",
			"parentId":      "$addUnit-14",
		}},
		Requires: []string{"addUnit-14"},
	}, {
		Id:     "addMachines-19",
		Method: "addMachines",
		Args: []interface{}{map[string]string{
			"containerType": "kvm",
			"parentId":      "$addUnit-15",
		}},
		Requires: []string{"addUnit-15"},
	}, {
		Id:       "addUnit-8",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addMachines-17"},
		Requires: []string{"addService-1", "addMachines-17"},
	}, {
		Id:       "addUnit-9",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addMachines-18"},
		Requires: []string{"addService-1", "addMachines-18"},
	}, {
		Id:       "addUnit-10",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addMachines-19"},
		Requires: []string{"addService-1", "addMachines-19"},
	}},
}, {
	about: "unit placed to machines",
	content: `
        services:
            django:
                charm: cs:trusty/django-42
                num_units: 5
                to:
                    - new
                    - 4
                    - kvm:8
                    - lxc:new
        machines:
            4:
                constraints: "cpu-cores=4"
            8:
                constraints: "cpu-cores=8"
    `,
	expected: []*bundlechanges.Change{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Args:   []interface{}{"cs:trusty/django-42"},
	}, {
		Id:     "addService-1",
		Method: "deploy",
		Args: []interface{}{
			"cs:trusty/django-42",
			"django",
			map[string]interface{}{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "addMachines-2",
		Method: "addMachines",
		Args: []interface{}{map[string]string{
			"series":      "",
			"constraints": "cpu-cores=4",
		}},
	}, {
		Id:     "addMachines-3",
		Method: "addMachines",
		Args: []interface{}{map[string]string{
			"series":      "",
			"constraints": "cpu-cores=8",
		}},
	}, {
		Id:       "addUnit-5",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addMachines-2"},
		Requires: []string{"addService-1", "addMachines-2"},
	}, {
		Id:     "addMachines-9",
		Method: "addMachines",
	}, {
		Id:     "addMachines-10",
		Method: "addMachines",
		Args: []interface{}{map[string]string{
			"containerType": "kvm",
			"parentId":      "$addMachines-3",
		}},
		Requires: []string{"addMachines-3"},
	}, {
		Id:     "addMachines-11",
		Method: "addMachines",
		Args:   []interface{}{map[string]string{"containerType": "lxc"}},
	}, {
		Id:     "addMachines-12",
		Method: "addMachines",
		Args:   []interface{}{map[string]string{"containerType": "lxc"}},
	}, {
		Id:       "addUnit-4",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addMachines-9"},
		Requires: []string{"addService-1", "addMachines-9"},
	}, {
		Id:       "addUnit-6",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addMachines-10"},
		Requires: []string{"addService-1", "addMachines-10"},
	}, {
		Id:       "addUnit-7",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addMachines-11"},
		Requires: []string{"addService-1", "addMachines-11"},
	}, {
		Id:       "addUnit-8",
		Method:   "addUnit",
		Args:     []interface{}{"$addService-1", 1, "$addMachines-12"},
		Requires: []string{"addService-1", "addMachines-12"},
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
