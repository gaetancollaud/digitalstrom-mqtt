package digitalstrom_mqtt

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"github.com/gaetancollaud/digitalstrom-mqtt/digitalstrom"
)

type DigitalstromMqtt struct {
	config *config.Config
	client mqtt.Client
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("MQTT Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("MQTT  Connect lost: %v", err)
}

func New(config *config.Config) *DigitalstromMqtt {
	inst := new(DigitalstromMqtt)
	inst.config = config

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf(config.MqttUrl))
	opts.SetClientID("digitalstrom-mqtt")
	//opts.SetUsername("emqx")
	//opts.SetPassword("public")
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	inst.client = client

	return inst
}

func (dm *DigitalstromMqtt) ListenForDeviceStatus(changes chan digitalstrom.DeviceStatusChanged) {
	for event := range changes {
		dm.publishDevice(event)
	}
}

func (dm *DigitalstromMqtt) ListenForCircuitValues(changes chan digitalstrom.CircuitValueChanged) {
	for event := range changes {
		dm.publishCircuit(event)
	}
}

func (dm *DigitalstromMqtt) publishDevice(changed digitalstrom.DeviceStatusChanged) {
	topic := "digitalstrom/devices/" + changed.Device.Name + "/" + changed.Channel + "/status"

	dm.client.Publish(topic, 0, false, fmt.Sprintf("%.2f", changed.NewValue))
}

func (dm *DigitalstromMqtt) publishCircuit(changed digitalstrom.CircuitValueChanged) {
	//fmt.Println("Updating meter", changed.Circuit.Name, changed.ConsumptionW, changed.EnergyWs)

	if changed.ConsumptionW != -1 {
		topic := "digitalstrom/meters/" + changed.Circuit.Name + "/consumptionW"
		dm.client.Publish(topic, 0, false, fmt.Sprintf("%d", changed.ConsumptionW))
	}

	if changed.EnergyWs != -1 {
		topic := "digitalstrom/meters/" + changed.Circuit.Name + "/EnergyWs"
		dm.client.Publish(topic, 0, false, fmt.Sprintf("%d", changed.EnergyWs))
	}
}
