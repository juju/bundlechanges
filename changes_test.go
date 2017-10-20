// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package bundlechanges_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	jujutesting "github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/charm.v6-unstable"

	"github.com/juju/bundlechanges"
)

type changesSuite struct {
	jujutesting.IsolationSuite
}

var _ = gc.Suite(&changesSuite{})

func TestPackage(t *testing.T) {
	gc.TestingT(t)
}

// record holds expected information about the contents of a change value.
type record struct {
	Id       string
	Requires []string
	Method   string
	Params   interface{}
	GUIArgs  []interface{}
}

var fromDataTests = []struct {
	// about describes the test.
	about string
	// content is the YAML encoded bundle content.
	content string
	// expected holds the expected changes required to deploy the bundle.
	expected []record
}{{
	about: "minimal bundle",
	content: `
        services:
            django:
                charm: django
    `,
	expected: []record{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm: "django",
		},
		GUIArgs: []interface{}{"django", ""},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-0",
			Application: "django",
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			"",
			"django",
			map[string]interface{}{},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{},
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
                expose: true
                options:
                    debug: false
                annotations:
                    gui-x: "609"
                    gui-y: "-15"
                resources:
                    data: 3
            mysql:
                charm: cs:precise/mysql-28
                num_units: 1
                resources:
                  data: "./resources/data.tar"
        series: trusty
        relations:
            - - mediawiki:db
              - mysql:db
    `,
	expected: []record{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "cs:precise/mediawiki-10",
			Series: "precise",
		},
		GUIArgs: []interface{}{"cs:precise/mediawiki-10", "precise"},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-0",
			Application: "mediawiki",
			Series:      "precise",
			Options:     map[string]interface{}{"debug": false},
			Resources:   map[string]int{"data": 3},
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			"precise",
			"mediawiki",
			map[string]interface{}{"debug": false},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{"data": 3},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "expose-2",
		Method: "expose",
		Params: bundlechanges.ExposeParams{
			Application: "$deploy-1",
		},
		GUIArgs:  []interface{}{"$deploy-1"},
		Requires: []string{"deploy-1"},
	}, {
		Id:     "setAnnotations-3",
		Method: "setAnnotations",
		Params: bundlechanges.SetAnnotationsParams{
			Id:          "$deploy-1",
			EntityType:  bundlechanges.ApplicationType,
			Annotations: map[string]string{"gui-x": "609", "gui-y": "-15"},
		},
		GUIArgs: []interface{}{
			"$deploy-1",
			"application",
			map[string]string{"gui-x": "609", "gui-y": "-15"},
		},
		Requires: []string{"deploy-1"},
	}, {
		Id:     "addCharm-4",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "cs:precise/mysql-28",
			Series: "precise",
		},
		GUIArgs: []interface{}{"cs:precise/mysql-28", "precise"},
	}, {
		Id:     "deploy-5",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:          "$addCharm-4",
			Application:    "mysql",
			Series:         "precise",
			LocalResources: map[string]string{"data": "./resources/data.tar"},
		},
		GUIArgs: []interface{}{
			"$addCharm-4",
			"precise",
			"mysql",
			map[string]interface{}{},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-4"},
	}, {
		Id:     "addRelation-6",
		Method: "addRelation",
		Params: bundlechanges.AddRelationParams{
			Endpoint1: "$deploy-1:db",
			Endpoint2: "$deploy-5:db",
		},
		GUIArgs:  []interface{}{"$deploy-1:db", "$deploy-5:db"},
		Requires: []string{"deploy-1", "deploy-5"},
	}, {
		Id:     "addUnit-7",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
		},
		GUIArgs:  []interface{}{"$deploy-1", nil},
		Requires: []string{"deploy-1"},
	}, {
		Id:     "addUnit-8",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-5",
		},
		GUIArgs:  []interface{}{"$deploy-5", nil},
		Requires: []string{"deploy-5"},
	}},
}, {
	about: "same charm reused",
	content: `
        services:
            mediawiki:
                charm: precise/mediawiki-10
                num_units: 1
            otherwiki:
                charm: precise/mediawiki-10
    `,
	expected: []record{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "precise/mediawiki-10",
			Series: "precise",
		},
		GUIArgs: []interface{}{"precise/mediawiki-10", "precise"},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-0",
			Application: "mediawiki",
			Series:      "precise",
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			"precise",
			"mediawiki",
			map[string]interface{}{},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "deploy-2",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-0",
			Application: "otherwiki",
			Series:      "precise",
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			"precise",
			"otherwiki",
			map[string]interface{}{},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "addUnit-3",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
		},
		GUIArgs:  []interface{}{"$deploy-1", nil},
		Requires: []string{"deploy-1"},
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
                constraints: cpu-cores=4 cpu-power=42
            haproxy:
                charm: cs:trusty/haproxy-47
                num_units: 2
                expose: yes
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
	expected: []record{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "cs:trusty/django-42",
			Series: "trusty",
		},
		GUIArgs: []interface{}{"cs:trusty/django-42", "trusty"},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-0",
			Application: "django",
			Series:      "trusty",
			Constraints: "cpu-cores=4 cpu-power=42",
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			"trusty",
			"django",
			map[string]interface{}{},
			"cpu-cores=4 cpu-power=42",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "addCharm-2",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "cs:trusty/haproxy-47",
			Series: "trusty",
		},
		GUIArgs: []interface{}{"cs:trusty/haproxy-47", "trusty"},
	}, {
		Id:     "deploy-3",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-2",
			Application: "haproxy",
			Series:      "trusty",
			Options:     map[string]interface{}{"bad": "wolf", "number": 42.47},
		},
		GUIArgs: []interface{}{
			"$addCharm-2",
			"trusty",
			"haproxy",
			map[string]interface{}{"bad": "wolf", "number": 42.47},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-2"},
	}, {
		Id:     "expose-4",
		Method: "expose",
		Params: bundlechanges.ExposeParams{
			Application: "$deploy-3",
		},
		GUIArgs:  []interface{}{"$deploy-3"},
		Requires: []string{"deploy-3"},
	}, {
		Id:     "addMachines-5",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			Series: "trusty",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{Series: "trusty"},
		},
	}, {
		Id:     "addMachines-6",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{},
		},
	}, {
		Id:     "addUnit-7",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-5",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addMachines-5"},
		Requires: []string{"deploy-1", "addMachines-5"},
	}, {
		Id:     "addMachines-11",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			ContainerType: "lxc",
			Series:        "trusty",
			ParentId:      "$addMachines-6",
			Constraints:   "cpu-cores=4 cpu-power=42",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				ContainerType: "lxc",
				Series:        "trusty",
				ParentId:      "$addMachines-6",
				Constraints:   "cpu-cores=4 cpu-power=42",
			},
		},
		Requires: []string{"addMachines-6"},
	}, {
		Id:     "addMachines-12",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			ContainerType: "lxc",
			Series:        "trusty",
			ParentId:      "$addUnit-7",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				ContainerType: "lxc",
				Series:        "trusty",
				ParentId:      "$addUnit-7",
			},
		},
		Requires: []string{"addUnit-7"},
	}, {
		Id:     "addMachines-13",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			Series: "trusty",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				Series: "trusty",
			},
		},
	}, {
		Id:     "addUnit-8",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-11",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addMachines-11"},
		Requires: []string{"deploy-1", "addMachines-11", "addUnit-7"},
	}, {
		Id:     "addUnit-9",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-3",
			To:          "$addMachines-12",
		},
		GUIArgs:  []interface{}{"$deploy-3", "$addMachines-12"},
		Requires: []string{"deploy-3", "addMachines-12"},
	}, {
		Id:     "addUnit-10",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-3",
			To:          "$addMachines-13",
		},
		GUIArgs:  []interface{}{"$deploy-3", "$addMachines-13"},
		Requires: []string{"deploy-3", "addMachines-13", "addUnit-9"},
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
	expected: []record{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "cs:trusty/django-42",
			Series: "trusty",
		},
		GUIArgs: []interface{}{"cs:trusty/django-42", "trusty"},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-0",
			Application: "django",
			Series:      "trusty",
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			"trusty",
			"django",
			map[string]interface{}{},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "addMachines-2",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			Constraints: "cpu-cores=4",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				Constraints: "cpu-cores=4",
			},
		},
	}, {
		Id:     "setAnnotations-3",
		Method: "setAnnotations",
		Params: bundlechanges.SetAnnotationsParams{
			Id:          "$addMachines-2",
			EntityType:  bundlechanges.MachineType,
			Annotations: map[string]string{"foo": "bar"},
		},
		GUIArgs: []interface{}{
			"$addMachines-2",
			"machine",
			map[string]string{"foo": "bar"},
		},
		Requires: []string{"addMachines-2"},
	}, {
		Id:     "addUnit-4",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-2",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addMachines-2"},
		Requires: []string{"deploy-1", "addMachines-2"},
	}, {
		Id:     "addMachines-6",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			Series: "trusty",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				Series: "trusty",
			},
		},
	}, {
		Id:     "addUnit-5",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-6",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addMachines-6"},
		Requires: []string{"deploy-1", "addMachines-6", "addUnit-4"},
	}},
}, {
	about: "endpoint without relation name",
	content: `
        services:
            mediawiki:
                charm: cs:precise/mediawiki-10
            mysql:
                charm: cs:precise/mysql-28
                constraints: mem=42G
        relations:
            - - mediawiki:db
              - mysql
    `,
	expected: []record{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "cs:precise/mediawiki-10",
			Series: "precise",
		},
		GUIArgs: []interface{}{"cs:precise/mediawiki-10", "precise"},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-0",
			Application: "mediawiki",
			Series:      "precise",
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			"precise",
			"mediawiki",
			map[string]interface{}{},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "addCharm-2",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "cs:precise/mysql-28",
			Series: "precise",
		},
		GUIArgs: []interface{}{"cs:precise/mysql-28", "precise"},
	}, {
		Id:     "deploy-3",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-2",
			Application: "mysql",
			Series:      "precise",
			Constraints: "mem=42G",
		},
		GUIArgs: []interface{}{
			"$addCharm-2",
			"precise",
			"mysql",
			map[string]interface{}{},
			"mem=42G",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-2"},
	}, {
		Id:     "addRelation-4",
		Method: "addRelation",
		Params: bundlechanges.AddRelationParams{
			Endpoint1: "$deploy-1:db",
			Endpoint2: "$deploy-3",
		},
		GUIArgs:  []interface{}{"$deploy-1:db", "$deploy-3"},
		Requires: []string{"deploy-1", "deploy-3"},
	}},
}, {
	about: "unit placed in application",
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
	expected: []record{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "cs:trusty/django-42",
			Series: "trusty",
		},
		GUIArgs: []interface{}{"cs:trusty/django-42", "trusty"},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-0",
			Application: "django",
			Series:      "trusty",
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			"trusty",
			"django",
			map[string]interface{}{},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "addCharm-2",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm: "wordpress",
		},
		GUIArgs: []interface{}{"wordpress", ""},
	}, {
		Id:     "deploy-3",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-2",
			Application: "wordpress",
		},
		GUIArgs: []interface{}{
			"$addCharm-2",
			"",
			"wordpress",
			map[string]interface{}{},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-2"},
	}, {
		Id:     "addUnit-6",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-3",
		},
		GUIArgs:  []interface{}{"$deploy-3", nil},
		Requires: []string{"deploy-3"},
	}, {
		Id:     "addUnit-7",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-3",
		},
		GUIArgs:  []interface{}{"$deploy-3", nil},
		Requires: []string{"deploy-3", "addUnit-6"},
	}, {
		Id:     "addUnit-8",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-3",
		},
		GUIArgs:  []interface{}{"$deploy-3", nil},
		Requires: []string{"deploy-3", "addUnit-7"},
	}, {
		Id:     "addUnit-4",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addUnit-6",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addUnit-6"},
		Requires: []string{"deploy-1", "addUnit-6"},
	}, {
		Id:     "addUnit-5",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addUnit-7",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addUnit-7"},
		Requires: []string{"deploy-1", "addUnit-7", "addUnit-4"},
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
	expected: []record{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "cs:trusty/django-42",
			Series: "trusty",
		},
		GUIArgs: []interface{}{"cs:trusty/django-42", "trusty"},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-0",
			Application: "django",
			Series:      "trusty",
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			"trusty",
			"django",
			map[string]interface{}{},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "addCharm-2",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "cs:trusty/mem-47",
			Series: "trusty",
		},
		GUIArgs: []interface{}{"cs:trusty/mem-47", "trusty"},
	}, {
		Id:     "deploy-3",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-2",
			Application: "memcached",
			Series:      "trusty",
		},
		GUIArgs: []interface{}{
			"$addCharm-2",
			"trusty",
			"memcached",
			map[string]interface{}{},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-2"},
	}, {
		Id:     "addCharm-4",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "vivid/rails",
			Series: "vivid",
		},
		GUIArgs: []interface{}{"vivid/rails", "vivid"},
	}, {
		Id:     "deploy-5",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-4",
			Application: "ror",
			Series:      "vivid",
		},
		GUIArgs: []interface{}{
			"$addCharm-4",
			"vivid",
			"ror",
			map[string]interface{}{},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-4"},
	}, {
		Id:     "addMachines-6",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			Series: "trusty",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				Series:      "trusty",
				Constraints: "",
			},
		},
	}, {
		Id:     "addUnit-12",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-3",
			To:          "$addMachines-6",
		},
		GUIArgs:  []interface{}{"$deploy-3", "$addMachines-6"},
		Requires: []string{"deploy-3", "addMachines-6"},
	}, {
		Id:     "addMachines-17",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			Series: "trusty",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				Series: "trusty",
			},
		},
	}, {
		Id:     "addMachines-18",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			Series: "trusty",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				Series: "trusty",
			},
		},
	}, {
		Id:     "addMachines-19",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			Series: "vivid",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				Series: "vivid",
			},
		},
	}, {
		Id:     "addUnit-7",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addUnit-12",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addUnit-12"},
		Requires: []string{"deploy-1", "addUnit-12"},
	}, {
		Id:     "addUnit-13",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-3",
			To:          "$addMachines-17",
		},
		GUIArgs:  []interface{}{"$deploy-3", "$addMachines-17"},
		Requires: []string{"deploy-3", "addMachines-17", "addUnit-12"},
	}, {
		Id:     "addUnit-14",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-3",
			To:          "$addMachines-18",
		},
		GUIArgs:  []interface{}{"$deploy-3", "$addMachines-18"},
		Requires: []string{"deploy-3", "addMachines-18", "addUnit-13"},
	}, {
		Id:     "addUnit-15",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-5",
			To:          "$addMachines-19",
		},
		GUIArgs:  []interface{}{"$deploy-5", "$addMachines-19"},
		Requires: []string{"deploy-5", "addMachines-19"},
	}, {
		Id:     "addUnit-16",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-5",
			To:          "$addMachines-6",
		},
		GUIArgs:  []interface{}{"$deploy-5", "$addMachines-6"},
		Requires: []string{"deploy-5", "addMachines-6", "addUnit-15"},
	}, {
		Id:     "addMachines-20",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			ContainerType: "lxc",
			Series:        "trusty",
			ParentId:      "$addUnit-13",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				ContainerType: "lxc",
				Series:        "trusty",
				ParentId:      "$addUnit-13",
			},
		},
		Requires: []string{"addUnit-13"},
	}, {
		Id:     "addMachines-21",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			ContainerType: "lxc",
			Series:        "trusty",
			ParentId:      "$addUnit-14",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				ContainerType: "lxc",
				Series:        "trusty",
				ParentId:      "$addUnit-14",
			},
		},
		Requires: []string{"addUnit-14"},
	}, {
		Id:     "addMachines-22",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			ContainerType: "kvm",
			Series:        "trusty",
			ParentId:      "$addUnit-15",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				ContainerType: "kvm",
				Series:        "trusty",
				ParentId:      "$addUnit-15",
			},
		},
		Requires: []string{"addUnit-15"},
	}, {
		Id:     "addMachines-23",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			ContainerType: "kvm",
			Series:        "trusty",
			ParentId:      "$addUnit-16",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				ContainerType: "kvm",
				Series:        "trusty",
				ParentId:      "$addUnit-16",
			},
		},
		Requires: []string{"addUnit-16"},
	}, {
		Id:     "addUnit-8",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-20",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addMachines-20"},
		Requires: []string{"deploy-1", "addMachines-20", "addUnit-7"},
	}, {
		Id:     "addUnit-9",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-21",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addMachines-21"},
		Requires: []string{"deploy-1", "addMachines-21", "addUnit-8"},
	}, {
		Id:     "addUnit-10",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-22",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addMachines-22"},
		Requires: []string{"deploy-1", "addMachines-22", "addUnit-9"},
	}, {
		Id:     "addUnit-11",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-23",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addMachines-23"},
		Requires: []string{"deploy-1", "addMachines-23", "addUnit-10"},
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
	expected: []record{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "cs:trusty/django-42",
			Series: "trusty",
		},
		GUIArgs: []interface{}{"cs:trusty/django-42", "trusty"},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-0",
			Application: "django",
			Series:      "trusty",
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			"trusty",
			"django",
			map[string]interface{}{},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "addMachines-2",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			Constraints: "cpu-cores=4",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				Constraints: "cpu-cores=4",
			},
		},
	}, {
		Id:     "addMachines-3",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			Constraints: "cpu-cores=8",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				Constraints: "cpu-cores=8",
			},
		},
	}, {
		Id:     "addMachines-9",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			Series: "trusty",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				Series: "trusty",
			},
		},
	}, {
		Id:     "addMachines-10",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			ContainerType: "kvm",
			Series:        "trusty",
			ParentId:      "$addMachines-3",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				ContainerType: "kvm",
				Series:        "trusty",
				ParentId:      "$addMachines-3",
			},
		},
		Requires: []string{"addMachines-3"},
	}, {
		Id:     "addMachines-11",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			ContainerType: "lxc",
			Series:        "trusty",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				ContainerType: "lxc",
				Series:        "trusty",
			},
		},
	}, {
		Id:     "addMachines-12",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			ContainerType: "lxc",
			Series:        "trusty",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				ContainerType: "lxc",
				Series:        "trusty",
			},
		},
	}, {
		Id:     "addUnit-4",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-9",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addMachines-9"},
		Requires: []string{"deploy-1", "addMachines-9"},
	}, {
		Id:     "addUnit-5",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-2",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addMachines-2"},
		Requires: []string{"deploy-1", "addMachines-2", "addUnit-4"},
	}, {
		Id:     "addUnit-6",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-10",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addMachines-10"},
		Requires: []string{"deploy-1", "addMachines-10", "addUnit-5"},
	}, {
		Id:     "addUnit-7",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-11",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addMachines-11"},
		Requires: []string{"deploy-1", "addMachines-11", "addUnit-6"},
	}, {
		Id:     "addUnit-8",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-12",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addMachines-12"},
		Requires: []string{"deploy-1", "addMachines-12", "addUnit-7"},
	}},
}, {
	about: "unit placed to new machine with constraints",
	content: `
        services:
            django:
                charm: cs:trusty/django-42
                num_units: 1
                to:
                    - new
                constraints: "cpu-cores=4"
    `,
	expected: []record{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "cs:trusty/django-42",
			Series: "trusty",
		},
		GUIArgs: []interface{}{"cs:trusty/django-42", "trusty"},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-0",
			Application: "django",
			Series:      "trusty",
			Constraints: "cpu-cores=4",
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			"trusty",
			"django",
			map[string]interface{}{},
			"cpu-cores=4",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "addMachines-3",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			Constraints: "cpu-cores=4",
			Series:      "trusty",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				Series:      "trusty",
				Constraints: "cpu-cores=4",
			},
		},
	}, {
		Id:     "addUnit-2",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-3",
		},
		GUIArgs:  []interface{}{"$deploy-1", "$addMachines-3"},
		Requires: []string{"deploy-1", "addMachines-3"},
	}},
}, {
	about: "application with storage",
	content: `
        services:
            django:
                charm: cs:trusty/django-42
                num_units: 2
                storage:
                    osd-devices: 3,30G
                    tmpfs: tmpfs,1G
    `,
	expected: []record{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "cs:trusty/django-42",
			Series: "trusty",
		},
		GUIArgs: []interface{}{"cs:trusty/django-42", "trusty"},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-0",
			Application: "django",
			Series:      "trusty",
			Storage: map[string]string{
				"osd-devices": "3,30G",
				"tmpfs":       "tmpfs,1G",
			},
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			"trusty",
			"django",
			map[string]interface{}{},
			"",
			map[string]string{
				"osd-devices": "3,30G",
				"tmpfs":       "tmpfs,1G",
			},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "addUnit-2",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
		},
		GUIArgs:  []interface{}{"$deploy-1", nil},
		Requires: []string{"deploy-1"},
	}, {
		Id:     "addUnit-3",
		Method: "addUnit",
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
		},
		GUIArgs:  []interface{}{"$deploy-1", nil},
		Requires: []string{"deploy-1", "addUnit-2"},
	}},
}, {
	about: "application with endpoint bindings",
	content: `
        services:
            django:
                charm: django
                bindings:
                    foo: bar
    `,
	expected: []record{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm: "django",
		},
		GUIArgs: []interface{}{"django", ""},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:            "$addCharm-0",
			Application:      "django",
			EndpointBindings: map[string]string{"foo": "bar"},
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			"",
			"django",
			map[string]interface{}{},
			"",
			map[string]string{},
			map[string]string{"foo": "bar"},
			map[string]int{},
		},
		Requires: []string{"addCharm-0"},
	}},
}, {
	about: "application with non-default series and placements ",
	content: `
series: trusty
services:
    gui3:
        charm: cs:precise/juju-gui
        num_units: 2
        to:
            - new
            - lxc:1
machines:
    1:
   `,
	expected: []record{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  "cs:precise/juju-gui",
			Series: "precise",
		},
		GUIArgs: []interface{}{"cs:precise/juju-gui", "precise"},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-0",
			Application: "gui3",
			Series:      "precise",
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			"precise",
			"gui3",
			map[string]interface{}{},
			"",
			map[string]string{},
			map[string]string{},
			map[string]int{},
		},
		Requires: []string{"addCharm-0"},
	}, {
		Id:     "addMachines-2",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			Series: "trusty",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				Series: "trusty",
			},
		},
	}, {
		Id:     "addMachines-5",
		Method: "addMachines",
		Params: bundlechanges.AddMachineParams{
			Series: "precise",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				Series: "precise",
			},
		},
	}, {
		Id:       "addMachines-6",
		Method:   "addMachines",
		Requires: []string{"addMachines-2"},
		Params: bundlechanges.AddMachineParams{
			ContainerType: "lxc",
			ParentId:      "$addMachines-2",
			Series:        "precise",
		},
		GUIArgs: []interface{}{
			bundlechanges.AddMachineOptions{
				ContainerType: "lxc",
				ParentId:      "$addMachines-2",
				Series:        "precise",
			},
		},
	}, {
		Id:       "addUnit-3",
		Method:   "addUnit",
		Requires: []string{"deploy-1", "addMachines-5"},
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-5",
		},
		GUIArgs: []interface{}{
			"$deploy-1",
			"$addMachines-5",
		},
	}, {
		Id:       "addUnit-4",
		Method:   "addUnit",
		Requires: []string{"deploy-1", "addMachines-6", "addUnit-3"},
		Params: bundlechanges.AddUnitParams{
			Application: "$deploy-1",
			To:          "$addMachines-6",
		},
		GUIArgs: []interface{}{
			"$deploy-1",
			"$addMachines-6",
		},
	}},
}}

func copyParams(value interface{}) interface{} {
	source := reflect.ValueOf(value).Elem().FieldByName("Params")
	target := reflect.New(source.Type()).Elem()

	for i := 0; i < source.NumField(); i++ {
		// Only copy public fields of the type.
		if targetField := target.Field(i); targetField.CanSet() {
			targetField.Set(source.Field(i))
		}
	}

	return target.Interface()
}

func (s *changesSuite) assertParseData(c *gc.C, content string, expected []record) {
	// Retrieve and validate the bundle data.
	data, err := charm.ReadBundleData(strings.NewReader(content))
	c.Assert(err, jc.ErrorIsNil)
	err = data.Verify(nil, nil)
	c.Assert(err, jc.ErrorIsNil)

	// Retrieve the changes, and convert them to a sequence of records.
	changes, err := bundlechanges.FromData(data, nil)
	c.Assert(err, jc.ErrorIsNil)
	records := make([]record, len(changes))
	for i, change := range changes {
		r := record{
			Id:       change.Id(),
			Requires: change.Requires(),
			Method:   change.Method(),
			GUIArgs:  change.GUIArgs(),
			Params:   copyParams(change),
		}
		records[i] = r
		c.Log(change.Description())
	}

	// Output the records for debugging.
	b, err := json.MarshalIndent(records, "", "  ")
	c.Assert(err, jc.ErrorIsNil)
	c.Logf("obtained records: %s", b)

	// Check that the obtained records are what we expect.
	c.Check(records, jc.DeepEquals, expected)
}

func (s *changesSuite) TestFromData(c *gc.C) {
	for i, test := range fromDataTests {
		c.Logf("\ntest %d: %s", i, test.about)
		s.assertParseData(c, test.content, test.expected)
	}
}

func (s *changesSuite) assertLocalBundleChanges(c *gc.C, charmDir, bundleContent, series string) {
	expected := []record{{
		Id:     "addCharm-0",
		Method: "addCharm",
		Params: bundlechanges.AddCharmParams{
			Charm:  charmDir,
			Series: series,
		},
		GUIArgs: []interface{}{charmDir, series},
	}, {
		Id:     "deploy-1",
		Method: "deploy",
		Params: bundlechanges.AddApplicationParams{
			Charm:       "$addCharm-0",
			Application: "django",
			Series:      series,
		},
		GUIArgs: []interface{}{
			"$addCharm-0",
			series,
			"django",
			map[string]interface{}{}, // options.
			"",                  // constraints.
			map[string]string{}, // storage.
			map[string]string{}, // endpoint bindings.
			map[string]int{},    // resources.
		},
		Requires: []string{"addCharm-0"},
	}}
	s.assertParseData(c, bundleContent, expected)
}

func (s *changesSuite) TestLocalCharmWithExplicitSeries(c *gc.C) {
	charmDir := c.MkDir()
	bundleContent := fmt.Sprintf(`
        services:
            django:
                charm: %s
                series: xenial
    `, charmDir)
	s.assertLocalBundleChanges(c, charmDir, bundleContent, "xenial")
}

func (s *changesSuite) TestLocalCharmWithSeriesFromCharm(c *gc.C) {
	charmDir := c.MkDir()
	bundleContent := fmt.Sprintf(`
        services:
            django:
                charm: %s
    `, charmDir)
	charmMeta := `
name: multi-series
summary: "That's a dummy charm with multi-series."
description: |
    This is a longer description which
    potentially contains multiple lines.
series:
    - precise
    - trusty
`[1:]
	err := ioutil.WriteFile(filepath.Join(charmDir, "metadata.yaml"), []byte(charmMeta), 0644)
	c.Assert(err, jc.ErrorIsNil)
	s.assertLocalBundleChanges(c, charmDir, bundleContent, "precise")
}

func (s *changesSuite) TestChangeDescriptions(c *gc.C) {
	for i, test := range []struct {
		description     string
		bundleContent   string
		existingModel   *bundlechanges.Model
		expectedChanges []string
		errMatch        string
	}{
		{
			description: "simple bundle, empty model",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-4
                        expose: yes
                        num_units: 1
                        options:
                            key-1: value-1
                            key-2: value-2
                        annotations:
                            gui-x: "10"
                            gui-y: "50"
            `,
			expectedChanges: []string{
				"upload charm cs:django-4",
				"deploy application django using cs:django-4",
				"expose django",
				"set annotations for django",
				"add unit django/0 to new machine 0",
			},
		}, {
			description: "charm in use by another application",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-4
                        num_units: 1
                        expose: yes
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"other-app": &bundlechanges.Application{
						Charm: "cs:django-4",
					},
				},
			},
			expectedChanges: []string{
				"deploy application django using cs:django-4",
				"expose django",
				"add unit django/0 to new machine 0",
			},
		}, {
			description: "charm upgrade",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-6
                        num_units: 1
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"django": &bundlechanges.Application{
						Charm: "cs:django-4",
						Units: []bundlechanges.Unit{
							{"django/0", "0"},
						},
					},
				},
			},
			expectedChanges: []string{
				"upload charm cs:django-6",
				"upgrade django to use charm cs:django-6",
			},
		}, {
			description: "app exists with less units",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-4
                        num_units: 2
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"django": &bundlechanges.Application{
						Charm: "cs:django-4",
						Units: []bundlechanges.Unit{
							{"django/0", "0"},
						},
					},
				},
				Machines: map[string]*bundlechanges.Machine{
					// We don't actually look at the content of the machines
					// for this test, just the keys.
					"0": nil,
				},
			},
			expectedChanges: []string{
				"add unit django/1 to new machine 1",
			},
		}, {
			description: "new machine number higher, unit higher",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-4
                        num_units: 2
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"django": &bundlechanges.Application{
						Charm: "cs:django-4",
						Units: []bundlechanges.Unit{
							{"django/0", "0"},
						},
					},
				},
				Machines: map[string]*bundlechanges.Machine{
					// We don't actually look at the content of the machines
					// for this test, just the keys.
					"0": nil,
				},
				Sequence: map[string]int{
					"application-django": 2,
					"machine":            3,
				},
			},
			expectedChanges: []string{
				"add unit django/2 to new machine 3",
			},
		}, {
			description: "app with different constraints",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-4
                        constraints: cpu-cores=4 cpu-power=42
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"django": &bundlechanges.Application{
						Charm: "cs:django-4",
						Units: []bundlechanges.Unit{
							{"django/0", "0"},
						},
					},
				},
				ConstraintsEqual: func(string, string) bool {
					return false
				},
				Machines: map[string]*bundlechanges.Machine{
					// We don't actually look at the content of the machines
					// for this test, just the keys.
					"0": nil,
				},
			},
			expectedChanges: []string{
				`set constraints for django to "cpu-cores=4 cpu-power=42"`,
			},
		}, {
			description: "app exists with enough units",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-4
                        num_units: 2
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"django": &bundlechanges.Application{
						Charm: "cs:django-4",
						Units: []bundlechanges.Unit{
							{"django/0", "0"},
							{"django/1", "1"},
							{"django/2", "2"},
						},
					},
				},
			},
			expectedChanges: []string{},
		}, {
			description: "app exists with changed options and annotations",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-4
                        num_units: 1
                        options:
                            key-1: value-1
                            key-2: value-2
                        annotations:
                            gui-x: "10"
                            gui-y: "50"
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"django": &bundlechanges.Application{
						Charm: "cs:django-4",
						Options: map[string]interface{}{
							"key-1": "value-1",
							"key-2": "value-4",
							"key-3": "value-5",
						},
						Annotations: map[string]string{
							"gui-x": "10",
							"gui-y": "40",
						},
						Units: []bundlechanges.Unit{
							{"django/0", "0"},
						},
					},
				},
			},
			expectedChanges: []string{
				"set application options for django",
				"set annotations for django",
			},
		}, {
			description: "new machine annotations and placement",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-4
                        exposed: true
                        num_units: 1
                        to: [1]
                machines:
                    1:
                        annotations:
                            foo: "10"
                            bar: "50"
            `,
			expectedChanges: []string{
				"upload charm cs:django-4",
				"deploy application django using cs:django-4",
				"add new machine 0 (bundle machine 1)",
				"set annotations for new machine 0",
				"add unit django/0 to new machine 0",
			},
		}, {
			description: "final placement not reused if specifies machine",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-4
                        num_units: 2
                        to: [1]
                machines:
                    1:
            `,
			expectedChanges: []string{
				"upload charm cs:django-4",
				"deploy application django using cs:django-4",
				"add new machine 0 (bundle machine 1)",
				"add unit django/0 to new machine 0",
				// NOTE: new machine, not put on $1.
				"add unit django/1 to new machine 1",
			},
		}, {
			description: "final placement not reused if specifies unit",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-4
                        num_units: 1
                    nginx:
                        charm: cs:nginx
                        num_units: 2
                        to: ["django/0"]
            `,
			expectedChanges: []string{
				"upload charm cs:django-4",
				"deploy application django using cs:django-4",
				"upload charm cs:nginx",
				"deploy application nginx using cs:nginx",
				"add unit django/0 to new machine 0",
				"add unit nginx/0 to new machine 0 to satisfy [django/0]",
				// NOTE: new machine, not put on $0.
				"add unit nginx/1 to new machine 1",
			},
		}, {
			description: "unit placed next to other new unit on existing machine",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-4
                        num_units: 1
                        to: [1]
                    nginx:
                        charm: cs:nginx
                        num_units: 1
                        to: ["django/0"]
                machines:
                    1:
            `,
			existingModel: &bundlechanges.Model{
				Machines: map[string]*bundlechanges.Machine{
					"0": &bundlechanges.Machine{ID: "0"},
				},
				MachineMap: map[string]string{"1": "0"},
			},
			expectedChanges: []string{
				"upload charm cs:django-4",
				"deploy application django using cs:django-4",
				"upload charm cs:nginx",
				"deploy application nginx using cs:nginx",
				"add unit django/0 to existing machine 0",
				"add unit nginx/0 to existing machine 0 to satisfy [django/0]",
			},
		}, {
			description: "application placement, not enough units",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-4
                        num_units: 3
                    nginx:
                        charm: cs:nginx
                        num_units: 5
                        to: [django]
            `,
			expectedChanges: []string{
				"upload charm cs:django-4",
				"deploy application django using cs:django-4",
				"upload charm cs:nginx",
				"deploy application nginx using cs:nginx",
				"add unit django/0 to new machine 0",
				"add unit django/1 to new machine 1",
				"add unit django/2 to new machine 2",
				"add unit nginx/0 to new machine 0 to satisfy [django]",
				"add unit nginx/1 to new machine 1 to satisfy [django]",
				"add unit nginx/2 to new machine 2 to satisfy [django]",
				"add unit nginx/3 to new machine 3",
				"add unit nginx/4 to new machine 4",
			},
		}, {
			description: "application placement, some existing",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-4
                        num_units: 5
                    nginx:
                        charm: cs:nginx
                        num_units: 5
                        to: [django]
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"django": &bundlechanges.Application{
						Charm: "cs:django-4",
						Units: []bundlechanges.Unit{
							{"django/0", "0"},
							{"django/1", "1"},
							{"django/3", "3"},
						},
					},
				},
				Machines: map[string]*bundlechanges.Machine{
					// We don't actually look at the content of the machines
					// for this test, just the keys.
					"0": nil, "1": nil, "3": nil,
				},
			},
			expectedChanges: []string{
				"upload charm cs:nginx",
				"deploy application nginx using cs:nginx",
				"add unit django/4 to new machine 4",
				"add unit django/5 to new machine 5",
				"add unit nginx/0 to existing machine 0 to satisfy [django]",
				"add unit nginx/1 to existing machine 1 to satisfy [django]",
				"add unit nginx/2 to existing machine 3 to satisfy [django]",
				"add unit nginx/3 to new machine 4 to satisfy [django]",
				"add unit nginx/4 to new machine 5 to satisfy [django]",
			},
		}, {
			description: "application placement, some co-located",
			bundleContent: `
                applications:
                    django:
                        charm: cs:django-4
                        num_units: 5
                    nginx:
                        charm: cs:nginx
                        num_units: 5
                        to: [django]
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"django": &bundlechanges.Application{
						Charm: "cs:django-4",
						Units: []bundlechanges.Unit{
							{"django/0", "0"},
							{"django/1", "1"},
							{"django/3", "3"},
						},
					},
					"nginx": &bundlechanges.Application{
						Charm: "cs:nginx",
						Units: []bundlechanges.Unit{
							{"nginx/0", "0"},
							{"nginx/1", "1"},
							{"nginx/2", "4"},
						},
					},
				},
				Machines: map[string]*bundlechanges.Machine{
					// We don't actually look at the content of the machines
					// for this test, just the keys.
					"0": nil, "1": nil, "3": nil, "4": nil,
				},
			},
			expectedChanges: []string{
				"add unit django/4 to new machine 5",
				"add unit django/5 to new machine 6",
				"add unit nginx/3 to existing machine 3 to satisfy [django]",
				"add unit nginx/4 to new machine 5 to satisfy [django]",
			},
		}, {
			description: "weird unit deployed, no existing model",
			bundleContent: `
                applications:
                    mysql:
                        charm: cs:mysql
                        num_units: 3
                        # The first placement directive here is skipped because
                        # the existing model already has one unit.
                        to: [new, "lxd:0", "lxd:new"]
                    keystone:
                        charm: cs:keystone
                        num_units: 3
                        to: ["lxd:mysql"]
                machines:
                    0:
            `,
			expectedChanges: []string{
				"upload charm cs:keystone",
				"deploy application keystone using cs:keystone",
				"upload charm cs:mysql",
				"deploy application mysql using cs:mysql",
				"add new machine 0",
				"add new machine 1",
				"add lxd container 0/lxd/0 on new machine 0",
				"add lxd container 2/lxd/0 on new machine 2",
				"add unit mysql/0 to new machine 1",
				"add unit mysql/1 to 0/lxd/0",
				"add unit mysql/2 to 2/lxd/0",
				"add lxd container 1/lxd/0 on new machine 1",
				"add lxd container 0/lxd/1 on new machine 0",
				"add lxd container 2/lxd/1 on new machine 2",
				"add unit keystone/0 to 1/lxd/0 to satisfy [lxd:mysql]",
				"add unit keystone/1 to 0/lxd/1 to satisfy [lxd:mysql]",
				"add unit keystone/2 to 2/lxd/1 to satisfy [lxd:mysql]",
			},
		}, {
			description: "unit deployed, defined machine",
			bundleContent: `
                applications:
                    mysql:
                        charm: cs:mysql
                        num_units: 3
                        # The first placement directive here is skipped because
                        # the existing model already has one unit.
                        to: [new, "lxd:0", "lxd:new"]
                    keystone:
                        charm: cs:keystone
                        num_units: 3
                        to: ["lxd:mysql"]
                machines:
                    0:
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"mysql": &bundlechanges.Application{
						Charm: "cs:mysql",
						Units: []bundlechanges.Unit{
							{"mysql/0", "0/lxd/0"},
						},
					},
				},
				Machines: map[string]*bundlechanges.Machine{
					// We don't actually look at the content of the machines
					// for this test, just the keys.
					"0": nil, "0/lxd/0": nil,
				},
			},
			expectedChanges: []string{
				"upload charm cs:keystone",
				"deploy application keystone using cs:keystone",
				"add new machine 1 (bundle machine 0)",
				"add unit keystone/0 to 0/lxd/1 to satisfy [lxd:mysql]",
				"add lxd container 1/lxd/0 on new machine 1",
				"add lxd container 2/lxd/0 on new machine 2",
				"add unit mysql/1 to 1/lxd/0",
				"add unit mysql/2 to 2/lxd/0",
				"add lxd container 1/lxd/1 on new machine 1",
				"add lxd container 2/lxd/1 on new machine 2",
				"add unit keystone/1 to 1/lxd/1 to satisfy [lxd:mysql]",
				"add unit keystone/2 to 2/lxd/1 to satisfy [lxd:mysql]",
			},
		}, {
			description: "lxd container sequence",
			bundleContent: `
                applications:
                    mysql:
                        charm: cs:mysql
                        num_units: 1
                    keystone:
                        charm: cs:keystone
                        num_units: 1
                        to: ["lxd:mysql"]
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"mysql": &bundlechanges.Application{
						Charm: "cs:mysql",
						Units: []bundlechanges.Unit{
							{"mysql/0", "0/lxd/0"},
						},
					},
				},
				Machines: map[string]*bundlechanges.Machine{
					// We don't actually look at the content of the machines
					// for this test, just the keys.
					"0": nil, "0/lxd/0": nil,
				},
				Sequence: map[string]int{
					"application-mysql": 1,
					"machine":           1,
					"machine-0/lxd":     2,
				},
			},
			expectedChanges: []string{
				"upload charm cs:keystone",
				"deploy application keystone using cs:keystone",
				"add unit keystone/0 to 0/lxd/2 to satisfy [lxd:mysql]",
			},
		}, {
			description: "unit deployed, defined machine, map 0 to 0",
			bundleContent: `
                applications:
                    mysql:
                        charm: cs:mysql
                        num_units: 3
                        # The first placement directive here is skipped because
                        # the existing model already has one unit.
                        to: [new, "lxd:0", "lxd:new"]
                    keystone:
                        charm: cs:keystone
                        num_units: 3
                        to: ["lxd:mysql"]
                machines:
                    0:
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"mysql": &bundlechanges.Application{
						Charm: "cs:mysql",
						Units: []bundlechanges.Unit{
							{"mysql/0", "0/lxd/0"},
						},
					},
				},
				Machines: map[string]*bundlechanges.Machine{
					"0":       &bundlechanges.Machine{ID: "0"},
					"0/lxd/0": &bundlechanges.Machine{ID: "0/lxd/0"},
				},
				MachineMap: map[string]string{
					"0": "0",
				},
			},
			expectedChanges: []string{
				// NOTE: we have mapped machine zero in the bundle to the
				// existing machine zero, and the second unit of mysql is said
				// to be in a lxd container on machine zero. We don't try to
				// map the existing mysql on machine zero to this unit, but
				// instead map it to the first placement directive. Hence,
				// this weird resulting placement.
				"upload charm cs:keystone",
				"deploy application keystone using cs:keystone",
				"add unit keystone/0 to 0/lxd/2 to satisfy [lxd:mysql]",
				"add unit mysql/1 to 0/lxd/1",
				"add lxd container 1/lxd/0 on new machine 1",
				"add lxd container 0/lxd/3 on existing machine 0",
				// This is arguably wrong since we have already placed a keystone
				// in a lxd container on this machine.
				"add unit keystone/1 to 0/lxd/3 to satisfy [lxd:mysql]",
				"add unit mysql/2 to 1/lxd/0",
				"add lxd container 1/lxd/1 on new machine 1",
				"add unit keystone/2 to 1/lxd/1 to satisfy [lxd:mysql]",
			},
		}, {
			description: "machine map to existing machine, some deployed",
			bundleContent: `
                applications:
                    mysql:
                        charm: cs:mysql
                        num_units: 3
                        # The first placement directive here is skipped because
                        # the existing model already has one unit.
                        to: [new, "lxd:0", "lxd:new"]
                    keystone:
                        charm: cs:keystone
                        num_units: 3
                        to: ["lxd:mysql"]
                machines:
                    0:
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"mysql": &bundlechanges.Application{
						Charm: "cs:mysql",
						Units: []bundlechanges.Unit{
							{"mysql/0", "0/lxd/0"},
						},
					},
				},
				Machines: map[string]*bundlechanges.Machine{
					"0":       &bundlechanges.Machine{ID: "0"},
					"0/lxd/0": &bundlechanges.Machine{ID: "0/lxd/0"},
					"2":       &bundlechanges.Machine{ID: "2"},
					"2/lxd/0": &bundlechanges.Machine{ID: "2/lxd/0"},
				},
				MachineMap: map[string]string{
					"0": "2", // 0 in bundle is machine 2 in existing.
				},
			},
			expectedChanges: []string{
				"upload charm cs:keystone",
				"deploy application keystone using cs:keystone",
				"add unit keystone/0 to 0/lxd/1 to satisfy [lxd:mysql]",
				"add unit mysql/1 to 2/lxd/1",
				"add lxd container 3/lxd/0 on new machine 3",
				"add lxd container 2/lxd/2 on existing machine 2",
				"add unit keystone/1 to 2/lxd/2 to satisfy [lxd:mysql]",
				"add unit mysql/2 to 3/lxd/0",
				"add lxd container 3/lxd/1 on new machine 3",
				"add unit keystone/2 to 3/lxd/1 to satisfy [lxd:mysql]",
			},
		}, {
			description: "setting annotations for existing machine",
			bundleContent: `
                applications:
                    mysql:
                        charm: cs:mysql
                        num_units: 1
                        to: ["0"]
                machines:
                    0:
                        annotations:
                            key: value
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"mysql": &bundlechanges.Application{
						Charm: "cs:mysql",
						Units: []bundlechanges.Unit{
							{"mysql/0", "0/lxd/0"},
						},
					},
				},
				Machines: map[string]*bundlechanges.Machine{
					"0":       &bundlechanges.Machine{ID: "0"},
					"0/lxd/0": &bundlechanges.Machine{ID: "0/lxd/0"},
					"2":       &bundlechanges.Machine{ID: "2"},
				},
				MachineMap: map[string]string{
					"0": "2", // 0 in bundle is machine 2 in existing.
				},
			},
			expectedChanges: []string{
				"set annotations for existing machine 2",
			},
		}, {
			description: "test sibling containers",
			bundleContent: `
                applications:
                    mysql:
                        charm: cs:mysql
                        num_units: 3
                        to: ["lxd:new"]
                    keystone:
                        charm: cs:keystone
                        num_units: 3
                        to: ["lxd:mysql"]
            `,
			expectedChanges: []string{
				"upload charm cs:keystone",
				"deploy application keystone using cs:keystone",
				"upload charm cs:mysql",
				"deploy application mysql using cs:mysql",
				"add lxd container 0/lxd/0 on new machine 0",
				"add lxd container 1/lxd/0 on new machine 1",
				"add lxd container 2/lxd/0 on new machine 2",
				"add unit mysql/0 to 0/lxd/0",
				"add unit mysql/1 to 1/lxd/0",
				"add unit mysql/2 to 2/lxd/0",
				"add lxd container 0/lxd/1 on new machine 0",
				"add lxd container 1/lxd/1 on new machine 1",
				"add lxd container 2/lxd/1 on new machine 2",
				"add unit keystone/0 to 0/lxd/1 to satisfy [lxd:mysql]",
				"add unit keystone/1 to 1/lxd/1 to satisfy [lxd:mysql]",
				"add unit keystone/2 to 2/lxd/1 to satisfy [lxd:mysql]",
			},
		}, {
			description: "test colocation into a container when specifying machine (don't ask why)",
			bundleContent: `
                applications:
                    mysql:
                        charm: cs:mysql
                        num_units: 3
                        to: ["lxd:new"]
                    keystone:
                        charm: cs:keystone
                        num_units: 3
                        to: [mysql/0, mysql/1, mysql/2]
            `,
			expectedChanges: []string{
				"upload charm cs:keystone",
				"deploy application keystone using cs:keystone",
				"upload charm cs:mysql",
				"deploy application mysql using cs:mysql",
				"add lxd container 0/lxd/0 on new machine 0",
				"add lxd container 1/lxd/0 on new machine 1",
				"add lxd container 2/lxd/0 on new machine 2",
				"add unit mysql/0 to 0/lxd/0",
				"add unit mysql/1 to 1/lxd/0",
				"add unit mysql/2 to 2/lxd/0",
				"add unit keystone/0 to 0/lxd/0 to satisfy [mysql/0]",
				"add unit keystone/1 to 1/lxd/0 to satisfy [mysql/1]",
				"add unit keystone/2 to 2/lxd/0 to satisfy [mysql/2]",
			},
		}, {
			description: "test colocation into a container (don't ask why)",
			bundleContent: `
                applications:
                    mysql:
                        charm: cs:mysql
                        num_units: 3
                        to: ["lxd:new"]
                    keystone:
                        charm: cs:keystone
                        num_units: 3
                        to: ["mysql"]
            `,
			expectedChanges: []string{
				"upload charm cs:keystone",
				"deploy application keystone using cs:keystone",
				"upload charm cs:mysql",
				"deploy application mysql using cs:mysql",
				"add lxd container 0/lxd/0 on new machine 0",
				"add lxd container 1/lxd/0 on new machine 1",
				"add lxd container 2/lxd/0 on new machine 2",
				"add unit mysql/0 to 0/lxd/0",
				"add unit mysql/1 to 1/lxd/0",
				"add unit mysql/2 to 2/lxd/0",
				"add unit keystone/0 to 0/lxd/0 to satisfy [mysql]",
				"add unit keystone/1 to 1/lxd/0 to satisfy [mysql]",
				"add unit keystone/2 to 2/lxd/0 to satisfy [mysql]",
			},
		}, {
			description: "test placement descriptions for unit placement",
			bundleContent: `
                applications:
                    mysql:
                        charm: cs:mysql
                        num_units: 3
                    keystone:
                        charm: cs:keystone
                        num_units: 3
                        to: ["lxd:mysql/0", "lxd:mysql/1", "lxd:mysql/2"]
            `,
			expectedChanges: []string{
				"upload charm cs:keystone",
				"deploy application keystone using cs:keystone",
				"upload charm cs:mysql",
				"deploy application mysql using cs:mysql",
				"add unit mysql/0 to new machine 0",
				"add unit mysql/1 to new machine 1",
				"add unit mysql/2 to new machine 2",
				"add lxd container 0/lxd/0 on new machine 0",
				"add lxd container 1/lxd/0 on new machine 1",
				"add lxd container 2/lxd/0 on new machine 2",
				"add unit keystone/0 to 0/lxd/0 to satisfy [lxd:mysql/0]",
				"add unit keystone/1 to 1/lxd/0 to satisfy [lxd:mysql/1]",
				"add unit keystone/2 to 2/lxd/0 to satisfy [lxd:mysql/2]",
			},
		}, {
			description: "cyclic placement",
			bundleContent: `
                applications:
                    mysql:
                        charm: cs:mysql
                        num_units: 3
                        to: [new, "lxd:0", "lxd:keystone/2"]
                    keystone:
                        charm: cs:keystone
                        num_units: 3
                        to: ["lxd:mysql"]
                machines:
                    0:
            `,
			errMatch: "cycle in placement directives for: keystone, mysql",
		}, {
			description: "test placement descriptions for unit placement",
			bundleContent: `
                applications:
                    mediawiki:
                        charm: cs:precise/mediawiki-10
                        num_units: 1
                        expose: true
                        options:
                            debug: false
                        annotations:
                            gui-x: "609"
                            gui-y: "-15"
                        resources:
                            data: 3
                    mysql:
                        charm: cs:precise/mysql-28
                        num_units: 1
                        resources:
                          data: "./resources/data.tar"
                series: trusty
                relations:
                    - - mediawiki:db
                      - mysql:db
            `,
			expectedChanges: []string{
				"upload charm cs:precise/mediawiki-10 for series precise",
				"deploy application mediawiki on precise using cs:precise/mediawiki-10",
				"expose mediawiki",
				"set annotations for mediawiki",
				"upload charm cs:precise/mysql-28 for series precise",
				"deploy application mysql on precise using cs:precise/mysql-28",
				"add relation mediawiki:db - mysql:db",
				"add unit mediawiki/0 to new machine 0",
				"add unit mysql/0 to new machine 1",
			},
		}, {
			description: "test unit ordering",
			bundleContent: `
                applications:
                    memcached:
                        charm: cs:xenial/mem-47
                        num_units: 3
                        to: [1, 2, 3]
                    django:
                        charm: cs:xenial/django-42
                        num_units: 4
                        to:
                            - 1
                            - lxd:memcached
                    ror:
                        charm: rails
                        num_units: 3
                        to:
                            - 1
                            - kvm:3
                machines:
                    1:
                    2:
                    3:
            `,
			expectedChanges: []string{
				"upload charm cs:xenial/django-42 for series xenial",
				"deploy application django on xenial using cs:xenial/django-42",
				"upload charm cs:xenial/mem-47 for series xenial",
				"deploy application memcached on xenial using cs:xenial/mem-47",
				"upload charm rails",
				"deploy application ror using rails",
				"add new machine 0 (bundle machine 1)",
				"add new machine 1 (bundle machine 2)",
				"add new machine 2 (bundle machine 3)",
				"add unit django/0 to new machine 0",
				"add unit memcached/0 to new machine 0",
				"add unit memcached/1 to new machine 1",
				"add unit memcached/2 to new machine 2",
				"add unit ror/0 to new machine 0",
				"add kvm container 2/kvm/0 on new machine 2",
				"add lxd container 0/lxd/0 on new machine 0",
				"add lxd container 1/lxd/0 on new machine 1",
				"add lxd container 2/lxd/0 on new machine 2",
				"add unit django/1 to 0/lxd/0 to satisfy [lxd:memcached]",
				"add unit django/2 to 1/lxd/0 to satisfy [lxd:memcached]",
				"add unit django/3 to 2/lxd/0 to satisfy [lxd:memcached]",
				"add unit ror/1 to 2/kvm/0",
				"add unit ror/2 to new machine 3",
			},
		}, {
			description: "add unit to existing app",
			bundleContent: `
                applications:
                    mediawiki:
                        charm: cs:precise/mediawiki-10
                        num_units: 2
                    mysql:
                        charm: cs:precise/mysql-28
                        num_units: 1
                series: trusty
                relations:
                    - - mediawiki:db
                      - mysql:db
            `,
			existingModel: &bundlechanges.Model{
				Applications: map[string]*bundlechanges.Application{
					"mediawiki": &bundlechanges.Application{
						Charm: "cs:precise/mediawiki-10",
						Units: []bundlechanges.Unit{
							{"mediawiki/0", "1"},
						},
					},
					"mysql": &bundlechanges.Application{
						Charm: "cs:precise/mysql-28",
						Units: []bundlechanges.Unit{
							{"mysql/0", "0"},
						},
					},
				},
				Machines: map[string]*bundlechanges.Machine{
					"0": &bundlechanges.Machine{ID: "0"},
					"1": &bundlechanges.Machine{ID: "1"},
				},
				Relations: []bundlechanges.Relation{
					{
						App1:      "mediawiki",
						Endpoint1: "db",
						App2:      "mysql",
						Endpoint2: "db",
					},
				},
			},
			expectedChanges: []string{
				"add unit mediawiki/1 to new machine 2",
			},
		}, {
			description: "test placement referring to own application",
			bundleContent: `
                applications:
                    problem:
                        charm: cs:problem
                        num_units: 2
                        to: ["lxd:new", "lxd:problem/0"]
            `,
			errMatch: `cycle in placement directives for: problem`,
		},
	} {
		c.Logf("%d: %s", i, test.description)

		data, err := charm.ReadBundleData(strings.NewReader(test.bundleContent))
		c.Assert(err, jc.ErrorIsNil)
		err = data.Verify(nil, nil)
		c.Assert(err, jc.ErrorIsNil)

		// Retrieve the changes, and convert them to a sequence of records.
		changes, err := bundlechanges.FromData(data, test.existingModel)
		if test.errMatch != "" {
			c.Assert(err, gc.ErrorMatches, test.errMatch)
		} else {
			c.Assert(err, jc.ErrorIsNil)
			var obtained []string
			for _, change := range changes {
				c.Log(change.Description())
				c.Logf("  %s %v", change.Method(), change.GUIArgs())

				obtained = append(obtained, change.Description())
			}
			c.Check(obtained, jc.DeepEquals, test.expectedChanges)
		}
	}
}
