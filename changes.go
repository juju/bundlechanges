// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package bundlechanges

import (
	"fmt"

	"gopkg.in/juju/charm.v6-unstable"
)

// FromData generates and returns the list of changes required to deploy the
// given bundle data. The changes are sorted by requirements, so that they can
// be applied in order. The bundle data is assumed to be already verified.
func FromData(data *charm.BundleData) []Change {
	cs := &changeset{}
	addedServices := handleServices(cs.add, data.Services)
	addedMachines := handleMachines(cs.add, data.Machines)
	handleRelations(cs.add, data.Relations, addedServices)
	handleUnits(cs.add, data.Services, addedServices, addedMachines)
	return cs.sorted()
}

// Change holds a single change required to deploy a bundle.
type Change interface {
	// Id returns the unique identifier for this change.
	Id() string
	// Requires returns a list of dependencies for this change. Each dependency
	// is represented by the corresponding change id, and must be applied
	// before this change is applied.
	Requires() []string
	// Method returns the action to be performed to apply this change.
	Method() string
	// GUIArgs returns positional arguments to pass to the method, suitable for
	// being serialized and sent to the Juju GUI.
	GUIArgs() []interface{}
	// setId is used to set the identifier for the change.
	setId(string)
}

type changeInfo struct {
	id       string
	requires []string
	method   string
}

// Id implements Change.Id.
func (ch *changeInfo) Id() string {
	return ch.id
}

// Requires implements Change.Requires.
func (ch *changeInfo) Requires() []string {
	if ch.requires == nil {
		return make([]string, 0)
	}
	return ch.requires
}

// Method implements Change.Method.
func (ch *changeInfo) Method() string {
	return ch.method
}

// setId implements Change.setId.
func (ch *changeInfo) setId(id string) {
	ch.id = id
}

// newAddCharmChange creates a new change for adding a charm.
func newAddCharmChange(args AddCharmArgs, requires ...string) *AddCharmChange {
	return &AddCharmChange{
		changeInfo: changeInfo{
			requires: requires,
			method:   "addCharm",
		},
		Args: args,
	}
}

// AddCharmChange holds a change for adding a charm to the environment.
type AddCharmChange struct {
	changeInfo
	// Args holds parameters for adding a charm.
	Args AddCharmArgs
}

// GUIArgs implements Change.GUIArgs.
func (ch *AddCharmChange) GUIArgs() []interface{} {
	return []interface{}{ch.Args.Charm}
}

// AddCharmArgs holds parameters for adding a charm to the environment.
type AddCharmArgs struct {
	// Charm holds the URL of the charm to be added.
	Charm string
}

// newAddMachineChange creates a new change for adding a machine or container.
func newAddMachineChange(args AddMachineArgs, requires ...string) *AddMachineChange {
	return &AddMachineChange{
		changeInfo: changeInfo{
			requires: requires,
			method:   "addMachines",
		},
		Args: args,
	}
}

// AddMachineChange holds a change for adding a machine or container.
type AddMachineChange struct {
	changeInfo
	// Args holds parameters for adding a machine.
	Args AddMachineArgs
}

// GUIArgs implements Change.GUIArgs.
func (ch *AddMachineChange) GUIArgs() []interface{} {
	options := AddMachineOptions{
		Series:        ch.Args.Series,
		Constraints:   ch.Args.Constraints,
		ContainerType: ch.Args.ContainerType,
		ParentId:      ch.Args.ParentId,
	}
	return []interface{}{options}
}

// AddMachineOptions holds GUI options for adding a machine or container.
type AddMachineOptions struct {
	// Series holds the machine OS series.
	Series string `json:"series,omitempty"`
	// Constraints holds the machine constraints.
	Constraints string `json:"constraints,omitempty"`
	// ContainerType holds the machine container type (like "lxc" or "kvm").
	ContainerType string `json:"containerType,omitempty"`
	// ParentId holds the id of the parent machine.
	ParentId string `json:"parentId,omitempty"`
}

// AddMachineArgs holds parameters for adding a machine or container.
type AddMachineArgs struct {
	// Series holds the optional machine OS series.
	Series string
	// Constraints holds the optional machine constraints.
	Constraints string
	// ContainerType optionally holds the type of the container (for instance
	// ""lxc" or kvm"). It is not specified for top level machines.
	ContainerType string
	// ParentId optionally holds a placeholder pointing to another machine
	// change or to a unit change. This value is only specified in the case
	// this machine is a container, in which case also ContainerType is set.
	ParentId string
}

// newAddRelationChange creates a new change for adding a relation.
func newAddRelationChange(args AddRelationArgs, requires ...string) *AddRelationChange {
	return &AddRelationChange{
		changeInfo: changeInfo{
			requires: requires,
			method:   "addRelation",
		},
		Args: args,
	}
}

// AddRelationChange holds a change for adding a relation between two services.
type AddRelationChange struct {
	changeInfo
	// Args holds parameters for adding a relation.
	Args AddRelationArgs
}

// GUIArgs implements Change.GUIArgs.
func (ch *AddRelationChange) GUIArgs() []interface{} {
	return []interface{}{ch.Args.Endpoint1, ch.Args.Endpoint2}
}

// AddRelationArgs holds parameters for adding a relation between two services.
type AddRelationArgs struct {
	// Endpoint1 and Endpoint2 hold relation endpoints, like "$deploy-1:web" or
	// just "$deploy-1". The service part of the endpoint is always a
	// placeholder pointing to a service change.
	Endpoint1 string
	Endpoint2 string
}

// newAddServiceChange creates a new change for adding a service.
func newAddServiceChange(args AddServiceArgs, requires ...string) *AddServiceChange {
	return &AddServiceChange{
		changeInfo: changeInfo{
			requires: requires,
			method:   "deploy",
		},
		Args: args,
	}
}

// AddServiceChange holds a change for deploying a Juju service.
type AddServiceChange struct {
	changeInfo
	// Args holds parameters for adding a service.
	Args AddServiceArgs
}

// GUIArgs implements Change.GUIArgs.
func (ch *AddServiceChange) GUIArgs() []interface{} {
	options := ch.Args.Options
	if options == nil {
		options = make(map[string]interface{}, 0)
	}
	return []interface{}{ch.Args.Charm, ch.Args.Service, options}
}

// AddServiceArgs holds parameters for deploying a Juju service.
type AddServiceArgs struct {
	// Charm holds the URL of the charm to be used to deploy this service.
	Charm string
	// Service holds the service name.
	Service string
	// Options holds service options.
	Options map[string]interface{}
	// TODO frankban: add support for service constraints.
}

// newAddUnitChange creates a new change for adding a service unit.
func newAddUnitChange(args AddUnitArgs, requires ...string) *AddUnitChange {
	return &AddUnitChange{
		changeInfo: changeInfo{
			requires: requires,
			method:   "addUnit",
		},
		Args: args,
	}
}

// AddUnitChange holds a change for adding a service unit.
type AddUnitChange struct {
	changeInfo
	// Args holds parameters for adding a unit.
	Args AddUnitArgs
}

// GUIArgs implements Change.GUIArgs.
func (ch *AddUnitChange) GUIArgs() []interface{} {
	args := []interface{}{ch.Args.Service, 1, nil}
	if ch.Args.To != "" {
		args[2] = ch.Args.To
	}
	return args
}

// AddUnitArgs holds parameters for adding a service unit.
type AddUnitArgs struct {
	// Service holds the service placeholder name for which a unit is added.
	Service string
	// To holds the optional location where to add the unit, as a placeholder
	// pointing to another unit change or to a machine change.
	To string
}

// newSetAnnotationsChange creates a new change for setting annotations.
func newSetAnnotationsChange(args SetAnnotationsArgs, requires ...string) *SetAnnotationsChange {
	return &SetAnnotationsChange{
		changeInfo: changeInfo{
			requires: requires,
			method:   "setAnnotations",
		},
		Args: args,
	}
}

// SetAnnotationsChange holds a change for setting service and machine
// annotations.
type SetAnnotationsChange struct {
	changeInfo
	// Args holds parameters for setting annotations.
	Args SetAnnotationsArgs
}

// GUIArgs implements Change.GUIArgs.
func (ch *SetAnnotationsChange) GUIArgs() []interface{} {
	return []interface{}{ch.Args.Id, ch.Args.EntityType, ch.Args.Annotations}
}

// AddServiceArgs holds parameters for setting annotations.
type SetAnnotationsArgs struct {
	// Id is the placeholder for the service or machine change corresponding to
	// the entity to be annotated.
	Id string
	// EntityType holds the type of the entity, "service" or "machine".
	EntityType string
	// Annotations holds the annotations as key/value pairs.
	Annotations map[string]string
}

// changeset holds the list of changes returned by FromData.
type changeset struct {
	changes []Change
}

// add adds the given change to this change set.
func (cs *changeset) add(change Change) {
	change.setId(fmt.Sprintf("%s-%d", change.Method(), len(cs.changes)))
	cs.changes = append(cs.changes, change)
}

// sorted returns the changes sorted by requirements, required first.
func (cs *changeset) sorted() []Change {
	numChanges := len(cs.changes)
	records := make(map[string]bool, numChanges)
	sorted := make([]Change, 0, numChanges)
	changes := make([]Change, numChanges, numChanges*2)
	copy(changes, cs.changes)
mainloop:
	for len(changes) != 0 {
		// Note that all valid bundles have at least two changes
		// (add one charm and deploy one service).
		change := changes[0]
		changes = changes[1:]
		for _, r := range change.Requires() {
			if !records[r] {
				// This change requires a change which is not yet listed.
				// Push this change at the end of the list and retry later.
				changes = append(changes, change)
				continue mainloop
			}
		}
		records[change.Id()] = true
		sorted = append(sorted, change)
	}
	return sorted
}
