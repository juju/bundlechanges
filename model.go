// Copyright 2017 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package bundlechanges

import (
	"fmt"

	"gopkg.in/juju/charm.v6-unstable"
	"gopkg.in/juju/names.v2"

	"github.com/juju/utils"
	"github.com/juju/utils/set"
)

// Model represents the existing deployment if any.
type Model struct {
	Applications map[string]*Application
	Machines     map[string]*Machine
	Relations    []Relation

	// ConstraintsEqual is a function that is able to determine if two
	// string values defining constraints are equal. This is to avoid a
	// hard dependency on the juju constraints package.
	ConstraintsEqual func(string, string) bool

	// This is a mapping of
	MachineMap map[string]string
}

type Relation struct {
	App1      string
	Endpoint1 string
	App2      string
	Endpoint2 string
}

func (m *Model) HasRelation(App1, Endpoint1, App2, Endpoint2 string) bool {
	if m == nil {
		return false
	}
	for _, rel := range m.Relations {
		oneWay := Relation{
			App1: App1, Endpoint1: Endpoint1, App2: App2, Endpoint2: Endpoint2,
		}
		other := Relation{
			App1: App2, Endpoint1: Endpoint2, App2: App1, Endpoint2: Endpoint1,
		}
		if rel == oneWay || rel == other {
			return true
		}
	}
	return false
}

func topLevelMachine(machineID string) string {
	if !names.IsContainerMachine(machineID) {
		return machineID
	}
	tag := names.NewMachineTag(machineID)
	return topLevelMachine(tag.Parent().Id())
}

// InferMachineMap looks at all the machines defined in the bundle
// and ifers their mapping to the existing machine.
// This method assumes that the units of an application are sorted
// in the natural sort order, meaning we start at unit zero and work
// our way up the unit numbers.
func (m *Model) InferMachineMap(data *charm.BundleData) {
	if m == nil {
		return
	}
	if m.MachineMap == nil {
		m.MachineMap = make(map[string]string)
	}
mainloop:
	for id := range data.Machines {
		// The simplst case is where the user has specified a mapping
		// for us.
		if _, found := m.MachineMap[id]; found {
			continue
		}
		// Look for a unit placement directive that specifies the machine.
		for appName, app := range data.Applications {
			for index, to := range app.To {
				// Here we explicitly ignore the error return of the parse placement
				// as the bundle should have been fully validated by now, which does
				// check the placement. However we do check to make sure the placement
				// is not nil (which it would be in an error case), because we don't
				// want to panic if for some weird reason, it does error.
				placement, _ := charm.ParsePlacement(to)
				if placement == nil || placement.Machine != id {
					continue
				}

				// See if we have deployed this unit yet.
				deployed := m.Applications[appName]
				if deployed == nil {
					continue
				}

				if len(deployed.Units) <= index {
					continue
				}

				unit := deployed.Units[index]
				m.MachineMap[id] = topLevelMachine(unit.Machine)
				continue mainloop
			}
		}
	}
}

// BundleMachine will return a the existing machine for the specified bundle
// amchine ID. If there is not a mapping available, nil is returned.
func (m *Model) BundleMachine(id string) *Machine {
	if m == nil || m.Machines == nil {
		return nil
	}
	// If the id isn't specified in the machine map, the empty string
	// is returned. If the no existing machine maps to the machine id,
	// a nil is returned from the Machines map.
	return m.Machines[m.MachineMap[id]]
}

func (m *Model) getUnitMachine(appName string, index int) string {
	if m == nil || m.Applications == nil {
		return ""
	}
	app := m.Applications[appName]
	if app == nil {
		return ""
	}
	target := fmt.Sprintf("%s/%d", appName, index)
	for _, unit := range app.Units {
		if unit.Name == target {
			return unit.Machine
		}
	}
	return ""
}

// Application represents an existing charm deployed in the model.
type Application struct {
	Name        string
	Charm       string // The charm URL.
	Options     map[string]interface{}
	Annotations map[string]string
	Constraints string // TODO: not updated yet.
	Exposed     bool
	// TODO: handle changes in:
	//   endpoint bindings -- possible even?
	//   storage
	//   series

	Units []Unit
}

type Unit struct {
	Name    string
	Machine string
}

// Machine represents an existing machine in the model.
type Machine struct {
	ID          string
	Annotations map[string]string
}

func (m *Model) hasCharm(charm string) bool {
	if m == nil || len(m.Applications) == 0 {
		return false
	}
	for _, app := range m.Applications {
		if app.Charm == charm {
			return true
		}
	}
	return false
}

func (m *Model) getApplication(name string) *Application {
	if m == nil {
		return nil
	}
	return m.Applications[name]
}

func (m *Model) unitMachinesWithoutApp(sourceApp, targetApp, container string) []string {
	source := m.getApplication(sourceApp)
	if source == nil {
		return []string{}
	}

	target := m.getApplication(targetApp)
	machines := set.NewStrings()
	for _, unit := range source.Units {
		machines.Add(topLevelMachine(unit.Machine))
	}
	if target != nil {
		for _, unit := range target.Units {
			if container == "" {
				machines.Remove(unit.Machine)
			} else {
				machineTag := names.NewMachineTag(unit.Machine)
				if machineTag.ContainerType() == container {
					machines.Remove(machineTag.Parent().Id())
				}
			}
		}
	}

	return utils.SortStringsNaturally(machines.Values())
}

func (a *Application) unitCount() int {
	if a == nil {
		return 0
	}
	return len(a.Units)
}

func (a *Application) changedAnnotations(annotations map[string]string) map[string]string {
	if a == nil || len(a.Annotations) == 0 {
		return annotations
	}
	changes := make(map[string]string)
	for key, value := range annotations {
		current, found := a.Annotations[key]
		if !found || current != value {
			changes[key] = value
		}
	}
	return changes
}

func (a *Application) changedOptions(options map[string]interface{}) map[string]interface{} {
	if a == nil || len(a.Options) == 0 {
		return options
	}
	changes := make(map[string]interface{})
	for key, value := range options {
		current, found := a.Options[key]
		// options should have been validated by now to only contain comparable
		// types.
		if !found || current != value {
			changes[key] = value
		}
	}
	return changes
}

func (m *Machine) changedAnnotations(annotations map[string]string) map[string]string {
	if m == nil || len(m.Annotations) == 0 {
		return annotations
	}
	changes := make(map[string]string)
	for key, value := range annotations {
		current, found := m.Annotations[key]
		if !found || current != value {
			changes[key] = value
		}
	}
	return changes
}
