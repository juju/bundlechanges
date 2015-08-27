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
func FromData(data *charm.BundleData) []*Change {
	cs := &changeset{}
	addedServices := handleServices(cs.add, data.Services)
	addedMachines := handleMachines(cs.add, data.Machines)
	handleRelations(cs.add, data.Relations, addedServices)
	handleUnits(cs.add, data.Services, addedServices, addedMachines)
	return cs.sorted()
}

// Change holds a single change required to deploy a bundle.
type Change struct {
	// Id is the unique identifier for this change.
	Id string `json:"id"`
	// Method is the action to be performed to apply this change.
	Method string `json:"method"`
	// Args holds a list of arguments to pass to the method.
	Args []interface{} `json:"args"`
	// Requires holds a list of dependencies for this change. Each dependency
	// is represented by the corresponding change id, and must be applied
	// before this change is applied.
	Requires []string `json:"requires"`
}

// changeset holds the list of changes returned by FromData.
type changeset struct {
	changes []*Change
}

// add is an addChangeFunc that can be used to add a change to this change set.
func (cs *changeset) add(method string, requires []string, args ...interface{}) *Change {
	if args == nil {
		args = make([]interface{}, 0)
	}
	if requires == nil {
		requires = make([]string, 0)
	}
	// TODO frankban: current Python bundle lib includes this inconsistency;
	// break compatibility and just use addService?
	idPrefix := method
	if method == "deploy" {
		idPrefix = "addService"
	}
	change := &Change{
		Id:       fmt.Sprintf("%s-%d", idPrefix, len(cs.changes)),
		Method:   method,
		Args:     args,
		Requires: requires,
	}
	cs.changes = append(cs.changes, change)
	return change
}

// sorted returns the changes sorted by requirements, required first.
func (cs *changeset) sorted() []*Change {
	numChanges := len(cs.changes)
	records := make(map[string]bool, numChanges)
	sorted := make([]*Change, 0, numChanges)
	changes := make([]*Change, numChanges, numChanges*2)
	copy(changes, cs.changes)
mainloop:
	for len(changes) != 0 {
		// Note that all valid bundles have at least two changes
		// (add one charm and deploy one service).
		change := changes[0]
		changes = changes[1:]
		for _, r := range change.Requires {
			if !records[r] {
				// This change requires a change which is not yet listed.
				// Push this change at the end of the list and retry later.
				changes = append(changes, change)
				continue mainloop
			}
		}
		records[change.Id] = true
		sorted = append(sorted, change)
	}
	return sorted
}
