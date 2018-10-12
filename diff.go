// Copyright 2018 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package bundlechanges

import (
	"reflect"
	"sort"
	"strings"

	"github.com/juju/collections/set"
	"gopkg.in/juju/charm.v6"
)

// DiffSide represents one side of a bundle-model diff.
type DiffSide string

const (
	// None represents neither the bundle or model side (used when neither is missing).
	None DiffSide = ""

	// BundleSide represents the bundle side of a diff.
	BundleSide DiffSide = "bundle"

	// ModelSide represents the model side of a diff.
	ModelSide DiffSide = "model"
)

// DiffConfig provides the values and configuration needed to diff the
// bundle and model.
type DiffConfig struct {
	Bundle *charm.BundleData
	Model  *Model

	IncludeAnnotations bool
	Logger             Logger
}

// BuildDiff returns a BundleDiff with the differences between the
// passed in bundle and model.
func BuildDiff(config DiffConfig) (*BundleDiff, error) {
	differ := &differ{config: config}
	return differ.build()
}

type differ struct {
	config DiffConfig
}

func (d *differ) build() (*BundleDiff, error) {
	return &BundleDiff{
		Applications: d.diffApplications(),
		Machines:     d.diffMachines(),
		Relations:    d.diffRelations(),
		// TODO(bundlediff): diff series.
	}, nil
}

func (d *differ) diffApplications() map[string]*ApplicationDiff {
	// Collect applications from both sides.
	allApps := set.NewStrings()
	for app := range d.config.Bundle.Applications {
		allApps.Add(app)
	}
	for app := range d.config.Model.Applications {
		allApps.Add(app)
	}

	results := make(map[string]*ApplicationDiff)
	for _, name := range allApps.SortedValues() {
		diff := d.diffApplication(name)
		if diff != nil {
			results[name] = diff
		}
	}
	if len(results) == 0 {
		return nil
	}
	return results
}

func (d *differ) diffApplication(name string) *ApplicationDiff {
	bundle, found := d.config.Bundle.Applications[name]
	if !found {
		return &ApplicationDiff{Missing: BundleSide}
	}
	model, found := d.config.Model.Applications[name]
	if !found {
		return &ApplicationDiff{Missing: ModelSide}
	}
	result := &ApplicationDiff{
		Charm:       d.diffStrings(bundle.Charm, model.Charm),
		NumUnits:    d.diffInts(bundle.NumUnits, len(model.Units)),
		Expose:      d.diffBools(bundle.Expose, model.Exposed),
		Constraints: d.diffStrings(bundle.Constraints, model.Constraints),
		Options:     d.diffOptions(bundle.Options, model.Options),
		// TODO(bundlediff): series
	}

	if d.config.IncludeAnnotations {
		result.Annotations = d.diffAnnotations(bundle.Annotations, model.Annotations)
	}

	if result.Empty() {
		return nil
	}
	return result
}

func (d *differ) diffMachines() map[string]*MachineDiff {
	// Collect machines from both sides.
	allNames := set.NewStrings()
	for name := range d.config.Bundle.Machines {
		allNames.Add(name)
	}
	for name := range d.config.Model.Machines {
		allNames.Add(name)
	}

	results := make(map[string]*MachineDiff)
	for _, name := range allNames.SortedValues() {
		diff := d.diffMachine(name)
		if diff != nil {
			results[name] = diff
		}
	}
	if len(results) == 0 {
		return nil
	}
	return results
}

func (d *differ) diffMachine(name string) *MachineDiff {
	bundle, found := d.config.Bundle.Machines[name]
	if !found {
		return &MachineDiff{Missing: BundleSide}
	}
	if bundle == nil {
		// This is equivalent to an empty machine spec.
		bundle = &charm.MachineSpec{}
	}
	model, found := d.config.Model.Machines[name]
	if !found {
		return &MachineDiff{Missing: ModelSide}
	}
	// TODO(bundlediff): series
	result := &MachineDiff{}

	if d.config.IncludeAnnotations {
		result.Annotations = d.diffAnnotations(bundle.Annotations, model.Annotations)
	}

	if result.Empty() {
		return nil
	}
	return result
}

func (d *differ) diffRelations() *RelationsDiff {
	bundleSet := make(map[Relation]bool)
	for _, relation := range d.config.Bundle.Relations {
		bundleSet[relationFromEndpoints(relation)] = true
	}

	modelSet := make(map[Relation]bool)
	var modelExtra []Relation
	for _, original := range d.config.Model.Relations {
		relation := canonicalRelation(original)
		modelSet[relation] = true
		_, found := bundleSet[relation]
		if !found {
			modelExtra = append(modelExtra, relation)
		}
	}

	var bundleExtra []Relation
	for relation := range bundleSet {
		_, found := modelSet[relation]
		if !found {
			bundleExtra = append(bundleExtra, relation)
		}
	}

	if len(bundleExtra) == 0 && len(modelExtra) == 0 {
		return nil
	}

	sort.Slice(bundleExtra, relationLess(bundleExtra))
	sort.Slice(modelExtra, relationLess(modelExtra))
	return &RelationsDiff{
		BundleExtra: toRelationSlices(bundleExtra),
		ModelExtra:  toRelationSlices(modelExtra),
	}
}

func (d *differ) diffAnnotations(bundle, model map[string]string) map[string]StringDiff {
	all := set.NewStrings()
	for name := range bundle {
		all.Add(name)
	}
	for name := range model {
		all.Add(name)
	}
	result := make(map[string]StringDiff)
	for _, name := range all.Values() {
		bundleValue := bundle[name]
		modelValue := model[name]
		if bundleValue != modelValue {
			result[name] = StringDiff{
				Bundle: bundleValue,
				Model:  modelValue,
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func (d *differ) diffOptions(bundle, model map[string]interface{}) map[string]OptionDiff {
	all := set.NewStrings()
	for name := range bundle {
		all.Add(name)
	}
	for name := range model {
		all.Add(name)
	}
	result := make(map[string]OptionDiff)
	for _, name := range all.Values() {
		bundleValue := bundle[name]
		modelValue := model[name]
		if !reflect.DeepEqual(bundleValue, modelValue) {
			result[name] = OptionDiff{
				Bundle: bundleValue,
				Model:  modelValue,
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func (d *differ) diffStrings(bundle, model string) *StringDiff {
	if bundle == model {
		return nil
	}
	return &StringDiff{Bundle: bundle, Model: model}
}

func (d *differ) diffInts(bundle, model int) *IntDiff {
	if bundle == model {
		return nil
	}
	return &IntDiff{Bundle: bundle, Model: model}
}

func (d *differ) diffBools(bundle, model bool) *BoolDiff {
	if bundle == model {
		return nil
	}
	return &BoolDiff{Bundle: bundle, Model: model}
}

func (d *differ) log(message string, args ...interface{}) {
	d.config.Logger.Tracef(message, args...)
}

// BundleDiff stores differences between a bundle and a model.
type BundleDiff struct {
	Applications map[string]*ApplicationDiff `yaml:"applications,omitempty"`
	Machines     map[string]*MachineDiff     `yaml:"machines,omitempty"`
	Series       *StringDiff                 `yaml:"series,omitempty"`
	Relations    *RelationsDiff              `yaml:"relations,omitempty"`
}

// Empty returns whether the compared bundle and model match (at least
// in terms of the details we check).
func (d *BundleDiff) Empty() bool {
	return len(d.Applications) == 0 &&
		len(d.Machines) == 0 &&
		d.Series == nil &&
		d.Relations == nil
}

// ApplicationDiff stores differences between an application in a bundle and a model.
type ApplicationDiff struct {
	Missing     DiffSide              `yaml:"missing,omitempty"`
	Charm       *StringDiff           `yaml:"charm,omitempty"`
	Series      *StringDiff           `yaml:"series,omitempty"`
	NumUnits    *IntDiff              `yaml:"num_units,omitempty"`
	Expose      *BoolDiff             `yaml:"expose,omitempty"`
	Options     map[string]OptionDiff `yaml:"options,omitempty"`
	Annotations map[string]StringDiff `yaml:"annotations,omitempty"`
	Constraints *StringDiff           `yaml:"constraints,omitempty"`

	// TODO (bundlediff): resources, storage, devices, endpoint
	// bindings
}

// Empty returns whether the compared bundle and model applications
// match.
func (d *ApplicationDiff) Empty() bool {
	return d.Missing == None &&
		d.Charm == nil &&
		d.Series == nil &&
		d.NumUnits == nil &&
		d.Expose == nil &&
		len(d.Options) == 0 &&
		len(d.Annotations) == 0 &&
		d.Constraints == nil
}

// StringDiff stores different bundle and model values for some
// string.
type StringDiff struct {
	Bundle string `yaml:"bundle"`
	Model  string `yaml:"model"`
}

// IntDiff stores different bundle and model values for some int.
type IntDiff struct {
	Bundle int `yaml:"bundle"`
	Model  int `yaml:"model"`
}

// BoolDiff stores different bundle and model values for some bool.
type BoolDiff struct {
	Bundle bool `yaml:"bundle"`
	Model  bool `yaml:"model"`
}

// OptionDiff stores different bundle and model values for some
// configuration value.
type OptionDiff struct {
	Bundle interface{} `yaml:"bundle"`
	Model  interface{} `yaml:"model"`
}

// MachineDiff stores differences between a machine in a bundle and a model.
type MachineDiff struct {
	Missing     DiffSide              `yaml:"missing,omitempty"`
	Annotations map[string]StringDiff `yaml:"annotations,omitempty"`
	Series      *StringDiff           `yaml:"series,omitempty"`
}

// Empty returns whether the compared bundle and model machines match.
func (d *MachineDiff) Empty() bool {
	return d.Missing == None &&
		len(d.Annotations) == 0 &&
		d.Series == nil
}

// RelationsDiff stores differences between relations in a bundle and
// model.
type RelationsDiff struct {
	BundleExtra [][]string `yaml:"bundle-extra,omitempty"`
	ModelExtra  [][]string `yaml:"model-extra,omitempty"`
}

// relationFromEndpoints returns a (canonicalised) Relation from a
// [app1:ep1 app2:ep2] bundle relation.
func relationFromEndpoints(relation []string) Relation {
	// TODO(bundlediff): verify bundle before we begin so we can rely
	// on the relations always being 2 app:endpoint strings.
	relation = relation[:]
	sort.Strings(relation)
	parts1 := strings.SplitN(relation[0], ":", 2)
	parts2 := strings.SplitN(relation[1], ":", 2)
	return Relation{
		App1:      parts1[0],
		Endpoint1: parts1[1],
		App2:      parts2[0],
		Endpoint2: parts2[1],
	}
}

// canonicalRelation ensures that the endpoints of the relation are in
// lexicographic order so we can put them into a map and find them
// even a relation was given to us in the other order.
func canonicalRelation(relation Relation) Relation {
	if relation.App1 < relation.App2 {
		return relation
	}
	if relation.App1 == relation.App2 && relation.Endpoint1 <= relation.Endpoint2 {
		return relation
	}
	// The endpoints need to be swapped.
	return Relation{
		App1:      relation.App2,
		Endpoint1: relation.Endpoint2,
		App2:      relation.App1,
		Endpoint2: relation.Endpoint1,
	}
}

// relationLess returns a func that compares Relations
// lexicographically.
func relationLess(relations []Relation) func(i, j int) bool {
	return func(i, j int) bool {
		a := relations[i]
		b := relations[j]
		if a.App1 < b.App1 {
			return true
		}
		if a.App1 > b.App1 {
			return false
		}
		if a.Endpoint1 < b.Endpoint1 {
			return true
		}
		if a.Endpoint1 > b.Endpoint1 {
			return false
		}
		if a.App2 < b.App2 {
			return true
		}
		if a.App2 > b.App2 {
			return false
		}
		return a.Endpoint2 < b.Endpoint2
	}
}

// toRelationSlices converts []Relation to [][]string{{"app:ep",
// "app:ep"}}.
func toRelationSlices(relations []Relation) [][]string {
	result := make([][]string, len(relations))
	for i, relation := range relations {
		result[i] = []string{
			relation.App1 + ":" + relation.Endpoint1,
			relation.App2 + ":" + relation.Endpoint2,
		}
	}
	return result
}
