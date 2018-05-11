// Copyright 2017 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package bundlechanges

import (
	"fmt"
	"strconv"

	"github.com/juju/naturalsort"
	"github.com/juju/utils"
	"github.com/juju/utils/set"
	"gopkg.in/juju/charm.v6"
	"gopkg.in/juju/names.v2"
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

	// Sequence holds a map of names to the next "number" that relates
	// to the unit or machine. The keys are "application-<name>", the string
	// "machine", or "machine-id/c" where n is a machine id, and c is a
	// container type.
	Sequence map[string]int

	// The Sequence map isn't touched during the processing of of bundle
	// changes, but we need to keep track, so a copy is made.
	sequence map[string]int

	// This is a mapping of existing machines to machines in the bundle.
	MachineMap map[string]string
}

type Relation struct {
	App1      string
	Endpoint1 string
	App2      string
	Endpoint2 string
}

func (m *Model) initializeSequence() {
	m.sequence = make(map[string]int)
	if m.Sequence != nil {
		for key, value := range m.Sequence {
			m.sequence[key] = value
		}
		// We assume that if the mapping was specified, a complete mapping was
		// specified.
		return
	}
	// Work to infer the mapping.

	for appName, app := range m.Applications {
		for _, unit := range app.Units {
			// This is pure paranoia, to avoid panics.
			if !names.IsValidUnit(unit.Name) {
				continue
			}
			u := names.NewUnitTag(unit.Name)
			unitNumber := u.Number()
			key := "application-" + appName
			if existing := m.sequence[key]; existing <= unitNumber {
				m.sequence[key] = unitNumber + 1
			}
		}
	}

	for machineID, _ := range m.Machines {
		// Continued paranoia.
		if !names.IsValidMachine(machineID) {
			continue
		}
		tag := names.NewMachineTag(machineID)
		key := "machine"
		// We know that the child id is always a valid integer.
		n, _ := strconv.Atoi(tag.ChildId())
		if containerType := tag.ContainerType(); containerType != "" {
			key = "machine-" + tag.Parent().Id() + "/" + containerType
		}
		if existing := m.sequence[key]; existing <= n {
			m.sequence[key] = n + 1
		}
	}
}

func (m *Model) nextMachine() string {
	value := m.sequence["machine"]
	m.sequence["machine"] = value + 1
	return strconv.Itoa(value)
}

func (m *Model) nextContainer(parentID, containerType string) string {
	key := "machine-" + parentID + "/" + containerType
	value := m.sequence[key]
	m.sequence[key] = value + 1
	return fmt.Sprintf("%s/%s/%d", parentID, containerType, value)
}

func (m *Model) nextUnit(appName string) string {
	key := "application-" + appName
	value := m.sequence[key]
	m.sequence[key] = value + 1
	return fmt.Sprintf("%s/%d", appName, value)
}

func (m *Model) HasRelation(App1, Endpoint1, App2, Endpoint2 string) bool {
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
	if m.MachineMap == nil {
		m.MachineMap = make(map[string]string)
	}
	// NOTE: makes a copy of the placements for each app as we consume them
	// as we iterate first time to allow for second pass to consume
	// remaining placement directives
	initialMachines := set.NewStrings()
	for appName, app := range data.Applications {
		for _, to := range app.To {
			placement, _ := charm.ParsePlacement(to)
			if placement == nil || placement.Machine == "" {
				continue
			}
			// If this machine is mapped already, skip this one.
			machine := placement.Machine
			if _, ok := m.MachineMap[machine]; ok {
				continue
			}
			if m.machineHasApp(machine, appName, placement.ContainerType) {
				m.MachineMap[machine] = machine
				initialMachines.Add(machine)
			} else {
			}
		}
	}

	var ids []string
	for id := range data.Machines {
		ids = append(ids, id)
	}
	naturalsort.Sort(ids)

mainloop:
	for _, id := range ids {
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
				// Find the first unit that we have't already used.

				unit := deployed.Units[index]
				machine := topLevelMachine(unit.Machine)
				if initialMachines.Contains(machine) {
					// Can't match the same machine twice.
					continue
				}
				m.MachineMap[id] = machine
				continue mainloop
			}
		}
	}
}

// BundleMachine will return a the existing machine for the specified bundle
// amchine ID. If there is not a mapping available, nil is returned.
func (m *Model) BundleMachine(id string) *Machine {
	if m.Machines == nil {
		return nil
	}
	// If the id isn't specified in the machine map, the empty string
	// is returned. If the no existing machine maps to the machine id,
	// a nil is returned from the Machines map.
	return m.Machines[m.MachineMap[id]]
}

func (m *Model) getUnitMachine(appName string, index int) string {
	if m.Applications == nil {
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
	if len(m.Applications) == 0 {
		return false
	}
	for _, app := range m.Applications {
		if app.Charm == charm {
			return true
		}
	}
	return false
}

// GetApplication returns the application specified or nil
// if it doesn't have it.
func (m *Model) GetApplication(name string) *Application {
	return m.Applications[name]
}

func (m *Model) unitMachinesWithoutApp(sourceApp, targetApp, container string) []string {
	source := m.GetApplication(sourceApp)
	if source == nil {
		return []string{}
	}

	target := m.GetApplication(targetApp)
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
					machines.Remove(topLevelMachine(unit.Machine))
				}
			}
		}
	}

	return utils.SortStringsNaturally(machines.Values())
}

func (m *Model) unsatisfiedMachineAndUnitPlacements(sourceApp string, placements []string) []string {
	// Cases we care about here are machine or unit placement.
	source := m.GetApplication(sourceApp)
	if source == nil {
		// Return a copy of the slice.
		return append([]string(nil), placements...)
	}

	var result []string

	for _, value := range placements {
		p, _ := charm.ParsePlacement(value)
		switch {
		case p.Machine == "new":
			result = append(result, value)
		case p.Machine != "":
			if !m.machineHasApp(p.Machine, sourceApp, p.ContainerType) {
				result = append(result, value)
			}
		case p.Application != "" && p.Unit < 0:
			result = append(result, value)
		case p.Application != "":
			machine := m.getUnitMachine(p.Application, p.Unit)
			if machine == "" {
				// This is unsatisfied because we don't have that unit.
				result = append(result, value)
			} else if !m.machineHasApp(machine, sourceApp, p.ContainerType) {
				result = append(result, value)
			}
		}
	}
	return result
}

func (m *Model) machineHasApp(machine, appName, containerType string) bool {
	if mappedMachine, ok := m.MachineMap[machine]; ok {
		machine = mappedMachine
	}
	app := m.GetApplication(appName)
	if app == nil {
		return false
	}
	for _, u := range app.Units {
		machineTag := names.NewMachineTag(u.Machine)
		if containerType == "" {
			if machineTag.ContainerType() == "" && machineTag.Id() == machine {
				return true
			}
		} else {
			if machineTag.ContainerType() == containerType &&
				machineTag.Parent().Id() == machine {
				return true
			}
		}
	}
	return false
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
		// types. Here we assume that the options have the correct type, and the
		// existing options have possibly been passed through JSON serialization
		// which converts int values to floats.
		switch value.(type) {
		case int:
			// If the validation code has done its job, the option from the
			// model should be a number too.
			switch cv := current.(type) {
			case float64: // JSON encoding converts ints to floats.
				current = int(cv)
			}
		}
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
