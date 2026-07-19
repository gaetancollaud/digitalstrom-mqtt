package digitalstrom

import (
	"fmt"
	"sync"
	"testing"
)

type apartmentStatusClientStub struct {
	Client
	statuses []*ApartmentStatus
}

type constantApartmentStatusClientStub struct {
	Client
	status *ApartmentStatus
}

func (c *constantApartmentStatusClientStub) GetApartmentStatus() (*ApartmentStatus, error) {
	return c.status, nil
}

func (c *apartmentStatusClientStub) GetApartmentStatus() (*ApartmentStatus, error) {
	status := c.statuses[0]
	c.statuses = c.statuses[1:]
	return status, nil
}

func TestRegistryPublishesApartmentStatusChanges(t *testing.T) {
	oldTemperature := 20.0
	newTemperature := 21.0
	client := &apartmentStatusClientStub{statuses: []*ApartmentStatus{
		{Attributes: ApartmentStatusAttributes{Measurements: ApartmentMeasurements{Temperature: &oldTemperature}}},
		{Attributes: ApartmentStatusAttributes{Measurements: ApartmentMeasurements{Temperature: &newTemperature}}},
	}}
	registry := &registry{
		digitalstromClient:             client,
		deviceChangeCallbacks:          map[string]DeviceChangeCallback{},
		apartmentStatusChangeCallbacks: map[string]ApartmentStatusChangeCallback{},
	}

	if err := registry.updateApartmentStatusAndFireChangeEvents(); err != nil {
		t.Fatalf("expected initial status update: %v", err)
	}
	var callbackOld *ApartmentStatus
	var callbackNew *ApartmentStatus
	if err := registry.ApartmentStatusChangeSubscribe("test", func(oldStatus *ApartmentStatus, newStatus *ApartmentStatus) {
		callbackOld = oldStatus
		callbackNew = newStatus
	}); err != nil {
		t.Fatalf("expected callback subscription: %v", err)
	}
	if err := registry.updateApartmentStatusAndFireChangeEvents(); err != nil {
		t.Fatalf("expected second status update: %v", err)
	}

	if callbackOld == nil || callbackNew == nil {
		t.Fatal("expected apartment status callback")
	}
	assertFloatPointer(t, "old temperature", callbackOld.Attributes.Measurements.Temperature, oldTemperature)
	assertFloatPointer(t, "new temperature", callbackNew.Attributes.Measurements.Temperature, newTemperature)
	current, err := registry.GetApartmentStatus()
	if err != nil {
		t.Fatalf("expected current status: %v", err)
	}
	assertFloatPointer(t, "current temperature", current.Attributes.Measurements.Temperature, newTemperature)

	if err := registry.ApartmentStatusChangeUnsubscribe("test"); err != nil {
		t.Fatalf("expected callback unsubscribe: %v", err)
	}
}

func TestRegistryApartmentStatusCallbackCanUnsubscribe(t *testing.T) {
	status := &ApartmentStatus{}
	registry := &registry{
		digitalstromClient:             &constantApartmentStatusClientStub{status: status},
		deviceChangeCallbacks:          map[string]DeviceChangeCallback{},
		apartmentStatusChangeCallbacks: map[string]ApartmentStatusChangeCallback{},
		apartmentStatus:                status,
	}

	called := false
	var unsubscribeErr error
	if err := registry.ApartmentStatusChangeSubscribe("test", func(*ApartmentStatus, *ApartmentStatus) {
		called = true
		unsubscribeErr = registry.ApartmentStatusChangeUnsubscribe("test")
	}); err != nil {
		t.Fatalf("expected callback subscription: %v", err)
	}

	if err := registry.updateApartmentStatusAndFireChangeEvents(); err != nil {
		t.Fatalf("expected status update: %v", err)
	}
	if !called {
		t.Fatal("expected apartment status callback")
	}
	if unsubscribeErr != nil {
		t.Fatalf("expected callback to unsubscribe itself: %v", unsubscribeErr)
	}
}

func TestRegistryReturnsAllFunctionBlocksForDevice(t *testing.T) {
	registry := &registry{
		devicesLookup: map[string]Device{
			"device-1": {
				DeviceId:   "device-1",
				Attributes: DeviceAttributes{Submodules: []string{"submodule-1", "submodule-2"}},
			},
		},
		submoduleLookup: map[string]Submodule{
			"submodule-1": {Attributes: SubmoduleAttributes{FunctionBlocks: []string{"function-block-1"}}},
			"submodule-2": {Attributes: SubmoduleAttributes{FunctionBlocks: []string{"function-block-2"}}},
		},
		functionBlocksLookup: map[string]FunctionBlock{
			"function-block-1": {FunctionBlockId: "function-block-1"},
			"function-block-2": {FunctionBlockId: "function-block-2"},
		},
	}

	functionBlocks, err := registry.GetFunctionBlocksForDevice("device-1")
	if err != nil {
		t.Fatalf("expected function blocks: %v", err)
	}
	if len(functionBlocks) != 2 {
		t.Fatalf("expected two function blocks, got %#v", functionBlocks)
	}
	if functionBlocks[0].FunctionBlockId != "function-block-1" || functionBlocks[1].FunctionBlockId != "function-block-2" {
		t.Fatalf("unexpected function blocks: %#v", functionBlocks)
	}
}

func TestRegistrySingleFunctionBlockErrorsDescribeCardinality(t *testing.T) {
	tests := []struct {
		name           string
		functionBlocks []string
		expected       string
	}{
		{name: "none", expected: "No function block found for device device-1"},
		{name: "multiple", functionBlocks: []string{"function-block-1", "function-block-2"}, expected: "Multiple function blocks found for device device-1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := &registry{
				devicesLookup: map[string]Device{
					"device-1": {DeviceId: "device-1", Attributes: DeviceAttributes{Submodules: []string{"submodule-1"}}},
				},
				submoduleLookup: map[string]Submodule{
					"submodule-1": {Attributes: SubmoduleAttributes{FunctionBlocks: test.functionBlocks}},
				},
				functionBlocksLookup: map[string]FunctionBlock{
					"function-block-1": {FunctionBlockId: "function-block-1"},
					"function-block-2": {FunctionBlockId: "function-block-2"},
				},
			}

			_, err := registry.GetFunctionBlockForDevice("device-1")
			if err == nil || err.Error() != test.expected {
				t.Fatalf("expected %q, got %v", test.expected, err)
			}
		})
	}
}

func TestRegistryApartmentStatusCallbacksAreConcurrentSafe(t *testing.T) {
	status := &ApartmentStatus{}
	registry := &registry{
		digitalstromClient:             &constantApartmentStatusClientStub{status: status},
		deviceChangeCallbacks:          map[string]DeviceChangeCallback{},
		apartmentStatusChangeCallbacks: map[string]ApartmentStatusChangeCallback{},
		apartmentStatus:                status,
	}

	start := make(chan struct{})
	errors := make(chan error, 2)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		<-start
		for i := 0; i < 1000; i++ {
			id := fmt.Sprintf("subscriber-%d", i)
			if err := registry.ApartmentStatusChangeSubscribe(id, func(*ApartmentStatus, *ApartmentStatus) {}); err != nil {
				errors <- err
				return
			}
			if err := registry.ApartmentStatusChangeUnsubscribe(id); err != nil {
				errors <- err
				return
			}
		}
	}()
	go func() {
		defer workers.Done()
		<-start
		for i := 0; i < 1000; i++ {
			if err := registry.updateApartmentStatusAndFireChangeEvents(); err != nil {
				errors <- err
				return
			}
			if _, err := registry.GetApartmentStatus(); err != nil {
				errors <- err
				return
			}
		}
	}()

	close(start)
	workers.Wait()
	close(errors)
	for err := range errors {
		t.Fatalf("concurrent apartment status access failed: %v", err)
	}
}
