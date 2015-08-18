// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package bundlechanges

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/juju/charm.v6-unstable"
)

// FromData generates and returns the set of changes required to deploy the
// given bundle data. The bundle data is assumed to be already verified.
func FromData(data *charm.BundleData) []*Change {
	cs := &changeset{}
	addedServices := handleServices(cs.add, data.Services)
	addedMachines := handleMachines(cs.add, data.Machines)
	handleRelations(cs.add, data.Relations, addedServices)
	handleUnits(cs.add, data.Services, addedServices, addedMachines)
	return cs.changes
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

// addChangeFunc is used to add a change to a change set. The resulting change
// is also returned.
type addChangeFunc func(method string, requires []string, args ...interface{}) *Change

// handleServices populates the change set with "addCharm"/"addService" records.
// This function also handles adding service annotations.
func handleServices(add addChangeFunc, services map[string]*charm.ServiceSpec) map[string]string {
	charms := make(map[string]string, len(services))
	addedServices := make(map[string]string, len(services))
	// Iterate over the map using its sorted keys so that results are
	// deterministic and easier to test.
	names := make([]string, 0, len(services))
	for name, _ := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		service := services[name]
		// Add the addCharm record if one hasn't been added yet.
		if charms[service.Charm] == "" {
			change := add("addCharm", nil, service.Charm)
			charms[service.Charm] = change.Id
		}

		// Add the addService record for this service.
		options := service.Options
		if options == nil {
			options = make(map[string]interface{}, 0)
		}
		change := add("deploy", []string{charms[service.Charm]}, service.Charm, name, options)
		addedServices[name] = change.Id

		// Add service annotations.
		if len(service.Annotations) > 0 {
			add("setAnnotations", []string{change.Id}, "$"+change.Id, "service", service.Annotations)
		}
	}
	return addedServices
}

// handleMachines populates the change set with "addMachines" records.
// This function also handles adding machine annotations.
func handleMachines(add addChangeFunc, machines map[string]*charm.MachineSpec) map[string]string {
	addedMachines := make(map[string]string, len(machines))
	// Iterate over the map using its sorted keys so that results are
	// deterministic and easier to test.
	names := make([]string, 0, len(machines))
	for name, _ := range machines {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		machine := machines[name]
		if machine == nil {
			machine = &charm.MachineSpec{}
		}
		// Add the addMachines record for this machine.
		change := add("addMachines", nil, map[string]string{
			"series":      machine.Series,
			"constraints": machine.Constraints,
		})
		addedMachines[name] = change.Id

		// Add machine annotations.
		if len(machine.Annotations) > 0 {
			add("setAnnotations", []string{change.Id}, "$"+change.Id, "machine", machine.Annotations)
		}
	}
	return addedMachines
}

// handleRelations populates the change set with "addRelation" records.
func handleRelations(add addChangeFunc, relations [][]string, addedServices map[string]string) {
	for _, relation := range relations {
		// Add the addRelation record for this relation pair.
		args := make([]interface{}, 2)
		requires := make([]string, 2)
		for i, endpoint := range relation {
			ep := parseEndpoint(endpoint)
			service := addedServices[ep.service]
			requires[i] = service
			ep.service = service
			args[i] = ep.String()
		}
		add("addRelation", requires, args...)
	}
}

// handleUnits populates the change set with "addUnit" records.
// It also handles adding machine containers where to place units if required.
func handleUnits(add addChangeFunc, services map[string]*charm.ServiceSpec, addedServices, addedMachines map[string]string) {
	records := make(map[string]*Change)
	// Iterate over the map using its sorted keys so that results are
	// deterministic and easier to test.
	names := make([]string, 0, len(services))
	for name, _ := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	// Collect and add all unit changes. These records are likely to be
	// modified later in order to handle unit placement.
	for _, name := range names {
		service := services[name]
		for i := 0; i < service.NumUnits; i++ {
			addedService := addedServices[name]
			change := add("addUnit", []string{addedService}, "$"+addedService, 1, nil)
			records[fmt.Sprintf("%s/%d", name, i)] = change
		}
	}
	// Now handle unit placement for each added service unit.
	for _, name := range names {
		service := services[name]
		numPlaced := len(service.To)
		if numPlaced == 0 {
			// If there are no placement directives it means that either the
			// service has no units (in which case there is no need to
			// proceed), or the units are not placed (in which case there is no
			// need to modify the change already added above).
			continue
		}
		// servicePlacedUnits holds, for each service, the number of units of
		// the current service already placed to that service.
		servicePlacedUnits := make(map[string]int)
		// At this point we know that we have at least one placement directive.
		// Fill the other ones if required.
		lastPlacement := service.To[numPlaced-1]
		for i := 0; i < service.NumUnits; i++ {
			p := lastPlacement
			if i < numPlaced {
				p = service.To[i]
			}
			// Generate the changes required in order to place this unit, and
			// retrieve the identifier of the parent change.
			parentId := unitParent(add, p, records, addedMachines, servicePlacedUnits)
			// Retrieve and modify the original "addUnit" change to add the
			// new parent requirement and placement target.
			change := records[fmt.Sprintf("%s/%d", name, i)]
			change.Requires = append(change.Requires, parentId)
			change.Args[2] = "$" + parentId
		}
	}
}

func unitParent(add addChangeFunc, p string, records map[string]*Change, addedMachines map[string]string, servicePlacedUnits map[string]int) (parentId string) {
	placement, err := charm.ParsePlacement(p)
	if err != nil {
		// Since the bundle is already verified, this should never happen.
		panic(err)
	}
	if placement.Machine == "new" {
		// The unit is placed to a new machine.
		var args []interface{}
		if placement.ContainerType != "" {
			args = []interface{}{
				map[string]string{"containerType": placement.ContainerType},
			}
		}
		return add("addMachines", nil, args...).Id
	}
	if placement.Machine != "" {
		// The unit is placed to a machine declared in the bundle.
		parentId = addedMachines[placement.Machine]
		if placement.ContainerType != "" {
			parentId = addContainer(add, placement.ContainerType, parentId)
		}
		return parentId
	}
	// The unit is placed to another unit or to a service.
	number := placement.Unit
	if number == -1 {
		// The unit is placed to a service. Calculate the unit number to be
		// used for unit co-location.
		if n, ok := servicePlacedUnits[placement.Service]; ok {
			number = n + 1
		} else {
			number = 0
		}
		servicePlacedUnits[placement.Service] = number
	}
	otherUnit := fmt.Sprintf("%s/%d", placement.Service, number)
	parentId = records[otherUnit].Id
	if placement.ContainerType != "" {
		parentId = addContainer(add, placement.ContainerType, parentId)
	}
	return parentId
}

func addContainer(add addChangeFunc, containerType, parentId string) string {
	change := add("addMachines", []string{parentId}, map[string]string{
		"containerType": containerType,
		"parentId":      "$" + parentId,
	})
	return change.Id
}

// parseEndpoint creates an endpoint from its string representation.
func parseEndpoint(e string) *endpoint {
	parts := strings.SplitN(e, ":", 2)
	ep := &endpoint{
		service: parts[0],
	}
	if len(parts) == 2 {
		ep.relation = parts[1]
	}
	return ep
}

// endpoint holds a relation endpoint.
type endpoint struct {
	service  string
	relation string
}

// String returns the string representation of an endpoint.
func (ep endpoint) String() string {
	if ep.relation == "" {
		return ep.service
	}
	return fmt.Sprintf("%s:%s", ep.service, ep.relation)
}
