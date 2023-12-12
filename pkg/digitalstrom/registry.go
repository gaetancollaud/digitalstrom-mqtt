package digitalstrom

import (
	"errors"
	"github.com/rs/zerolog/log"
	"sync"
)

type DeviceChangeCallback func(deviceId string, outputId string, oldValue float64, newValue float64)

type Registry interface {
	Start() error

	Stop() error

	GetDevices() ([]Device, error)

	GetDevice(deviceId string) (Device, error)

	GetFunctionBlockForDevice(deviceId string) (FunctionBlock, error)

	GetOutputsOfDevice(deviceId string) ([]Output, error)
	GetOutputValuesOfDevice(deviceId string) ([]OutputValue, error)

	GetControllers() ([]Controller, error)
	GetControllerById(controllerId string) (Controller, error)
	GetMeterings() ([]Metering, error)

	DeviceChangeSubscribe(deviceId string, callback DeviceChangeCallback) error
	DeviceChangeUnsubscribe(deviceId string) error
}

type registry struct {
	digitalstromClient Client

	apartment       *Apartment
	apartmentStatus *ApartmentStatus
	meterings       *Meterings

	controllersLookup    map[string]Controller
	devicesLookup        map[string]Device
	submoduleLookup      map[string]Submodule
	functionBlocksLookup map[string]FunctionBlock

	deviceChangeCallbacks map[string]DeviceChangeCallback

	registryLoading sync.Mutex
}

func NewRegistry(digitalstromClient Client) Registry {
	return &registry{
		digitalstromClient:    digitalstromClient,
		deviceChangeCallbacks: make(map[string]DeviceChangeCallback),
	}
}

func (r *registry) Start() error {
	if err := r.updateApartment(); err != nil {
		return err
	}
	if err := r.updateMeterings(); err != nil {
		return err
	}
	err := r.updateApartmentStatusAndFireChangeEvents()
	if err != nil {
		return err
	}
	callback := func(notification WebsocketNotification) {
		// TODO handle structure changes
		if err := r.updateApartmentStatusAndFireChangeEvents(); err != nil {
			log.Err(err).Msg("Error updating apartment status")
		}
	}
	if err := r.digitalstromClient.NotificationSubscribe("registry", callback); err != nil {
		return err
	}
	return nil
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

func (r *registry) GetOutputValuesOfDevice(deviceId string) ([]OutputValue, error) {
	outputs := []OutputValue{}
	for _, device := range r.apartmentStatus.Included.Devices {
		if device.DeviceId == deviceId {
			for _, functionBlockValue := range device.Attributes.FunctionBlocks {
				for _, outputValue := range functionBlockValue.Outputs {
					outputs = append(outputs, outputValue)
				}
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

func (r *registry) GetControllers() ([]Controller, error) {
	return r.apartment.Included.Controllers, nil
}

func (r *registry) GetControllerById(controllerId string) (Controller, error) {
	controller, ok := r.controllersLookup[controllerId]
	if ok {
		return controller, nil
	}
	return Controller{}, errors.New("No controller found with id " + controllerId)
}

func (r *registry) GetMeterings() ([]Metering, error) {
	return r.meterings.Meterings, nil
}

func (r *registry) updateApartment() error {
	r.registryLoading.Lock()
	defer r.registryLoading.Unlock()

	apartment, err := r.digitalstromClient.GetApartment()
	if err != nil {
		return err
	}

	r.apartment = apartment

	r.controllersLookup = make(map[string]Controller)
	r.devicesLookup = make(map[string]Device)
	r.submoduleLookup = make(map[string]Submodule)
	r.functionBlocksLookup = make(map[string]FunctionBlock)

	// Create lookup tables for fast access.
	for _, controller := range apartment.Included.Controllers {
		r.controllersLookup[controller.ControllerId] = controller
	}
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

func (r *registry) updateMeterings() error {
	r.registryLoading.Lock()
	defer r.registryLoading.Unlock()

	meterings, err := r.digitalstromClient.GetMeterings()
	if err != nil {
		return err
	}

	r.meterings = meterings

	return nil
}

func (r *registry) DeviceChangeSubscribe(deviceId string, callback DeviceChangeCallback) error {
	_, exists := r.deviceChangeCallbacks[deviceId]
	if exists {
		return errors.New("Callback already registered for device " + deviceId)
	}
	r.deviceChangeCallbacks[deviceId] = callback
	return nil
}

func (r *registry) DeviceChangeUnsubscribe(deviceId string) error {
	_, exists := r.deviceChangeCallbacks[deviceId]
	if !exists {
		return errors.New("No callback registered for device " + deviceId)
	}
	delete(r.deviceChangeCallbacks, deviceId)
	return nil
}

func (r *registry) updateApartmentStatusAndFireChangeEvents() error {
	oldStatus := r.apartmentStatus
	newStatus, err := r.digitalstromClient.GetApartmentStatus()
	if err != nil {
		return err
	}
	r.apartmentStatus = newStatus

	if oldStatus != nil {
		// Check diff and broadcast events

		oldStatusLookup := make(map[string]map[string]OutputValue)
		for _, device := range oldStatus.Included.Devices {
			oldStatusLookup[device.DeviceId] = make(map[string]OutputValue)
			for _, functionBlock := range device.Attributes.FunctionBlocks {
				for _, output := range functionBlock.Outputs {
					oldStatusLookup[device.DeviceId][output.OutputId] = output
				}
			}
		}

		for _, device := range newStatus.Included.Devices {
			for _, functionBlock := range device.Attributes.FunctionBlocks {
				for _, newOutput := range functionBlock.Outputs {
					oldOutput := oldStatusLookup[device.DeviceId][newOutput.OutputId]
					if oldOutput.TargetValue != newOutput.TargetValue {
						log.Info().
							Str("DeviceId", device.DeviceId).
							Str("Output", newOutput.OutputId).
							Float64("oldValue", oldOutput.TargetValue).
							Float64("newValue", newOutput.TargetValue).
							Msg("Output value changed")

						callback, exists := r.deviceChangeCallbacks[device.DeviceId]
						if exists {
							callback(device.DeviceId, newOutput.OutputId, oldOutput.TargetValue, newOutput.TargetValue)
						}
					}
				}
			}
		}
	}
	return nil
}
