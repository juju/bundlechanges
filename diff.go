// Copyright 2018 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package bundlechanges

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

// NewBundleDiff creates a new empty BundleDiff.
func NewBundleDiff() *BundleDiff {
	return &BundleDiff{
		Applications: make(map[string]*ApplicationDiff),
		Machines:     make(map[string]*MachineDiff),
	}
}

// BundleDiff stores differences between a bundle and a model.
type BundleDiff struct {
	Applications map[string]*ApplicationDiff `yaml:"applications,omitempty"`
	Machines     map[string]*MachineDiff     `yaml:"machines,omitempty"`
	Series       *StringDiff                 `yaml:"series,omitempty"`
	Relations    *RelationDiff               `yaml:"relations,omitempty"`
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
	Constraints *StringDiff           `yaml:"constraints,omitempty"`
	Annotations map[string]StringDiff `yaml:"annotations,omitempty"`
	Series      *StringDiff           `yaml:"series,omitempty"`
}

// RelationDiff stores differences between relations in a bundle and
// model.
type RelationDiff struct {
	BundleExtra [][]string `yaml:"bundle-extra,omitempty"`
	ModelExtra  [][]string `yaml:"model-extra,omitempty"`
}
