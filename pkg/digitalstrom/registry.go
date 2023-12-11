package digitalstrom

import (
	"errors"
	"sync"
)

type Registry interface {
	Start() error

	Stop() error

	GetDevices() ([]Device, error)

	GetDevice(deviceId string) (Device, error)

	GetFunctionBlockForDevice(deviceId string) (FunctionBlock, error)

	GetOutputsOfDevice(deviceId string) ([]Output, error)

	GetMeterings() []Metering
}

type registry struct {
	digitalstromClient Client

	apartment *Apartment

	devicesLookup        map[string]Device
	submoduleLookup      map[string]Submodule
	functionBlocksLookup map[string]FunctionBlock

	registryLoading sync.Mutex
}

func NewRegistry(digitalstromClient Client) Registry {
	return &registry{
		digitalstromClient: digitalstromClient,
	}
}

func (r *registry) Start() error {
	return r.updateApartment()
}

func (r *registry) Stop() error {
	return nil
}

func (r *registry) GetDevices() ([]Device, error) {
	return r.apartment.Included.Devices, nil
}

func (r *registry) GetDevice(deviceId string) (Device, error) {
	device, ok := r.devicesLookup[deviceId]
	if ok {
		return device, nil
	}
	return Device{}, errors.New("No device found with id " + deviceId)
}

func (r *registry) GetOutputsOfDevice(deviceId string) ([]Output, error) {
	device, err := r.GetDevice(deviceId)
	if err != nil {
		return nil, err
	}

	outputs := []Output{}
	for _, submoduleId := range device.Attributes.Submodules {
		submodule := r.submoduleLookup[submoduleId]
		for _, functionBlockId := range submodule.Attributes.FunctionBlocks {
			functionBlock := r.functionBlocksLookup[functionBlockId]
			for _, output := range functionBlock.Attributes.Outputs {
				outputs = append(outputs, output)
			}
		}
	}

	return outputs, nil
}

func (r *registry) GetFunctionBlockForDevice(deviceId string) (FunctionBlock, error) {
	device, err := r.GetDevice(deviceId)
	if err != nil {
		return FunctionBlock{}, err
	}

	var functionBlocks []FunctionBlock

	for _, submoduleId := range device.Attributes.Submodules {
		submodule := r.submoduleLookup[submoduleId]
		for _, functionBlockId := range submodule.Attributes.FunctionBlocks {
			functionBlock := r.functionBlocksLookup[functionBlockId]
			functionBlocks = append(functionBlocks, functionBlock)
		}
	}

	length := len(functionBlocks)
	if length == 0 {
		return FunctionBlock{}, errors.New("Multiple function blocks found for device " + deviceId)
	}
	if length > 1 {
		return FunctionBlock{}, errors.New("No function block found for device " + deviceId)
	}
	return functionBlocks[0], nil
}

func (r *registry) GetMeterings() []Metering {
	return nil
}

func (r *registry) updateApartment() error {
	r.registryLoading.Lock()
	defer r.registryLoading.Unlock()

	apartment, err := r.digitalstromClient.GetApartment()
	if err != nil {
		return err
	}

	r.apartment = apartment

	r.devicesLookup = make(map[string]Device)
	r.submoduleLookup = make(map[string]Submodule)
	r.functionBlocksLookup = make(map[string]FunctionBlock)

	// Create lookup tables for fast access.
	for _, device := range apartment.Included.Devices {
		r.devicesLookup[device.DeviceId] = device
	}
	for _, submodule := range apartment.Included.Submodules {
		r.submoduleLookup[submodule.SubmoduleId] = submodule
	}
	for _, functionBlock := range apartment.Included.FunctionBlocks {
		r.functionBlocksLookup[functionBlock.FunctionBlockId] = functionBlock
	}

	return nil
}
