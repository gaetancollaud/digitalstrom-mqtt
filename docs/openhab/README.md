# Openhab

## MQTT configuration
*things/mqtt.things*

```
Bridge mqtt:broker:mosquitto [ host="192.168.1.12", port="1883" ]
{
    // Example of dSM12
    Thing topic ds_circuit_chambre "Circuit chambre" {
    Channels:
        Type number : energyWs     [ stateTopic="digitalstrom/circuits/chambres/EnergyWs/state"]
        Type number : consumptionW [ stateTopic="digitalstrom/circuits/chambres/consumptionW/state"]
    }
    
    // Example of light. Can be GE-KM200 or GE-TKM210 for example
    Thing topic ds_bas_gaetan_light "Lumière Gaétan" {
    Channels:
        Type dimmer : state     [ stateTopic="digitalstrom/devices/light_gaetan/brightness/state", commandTopic="digitalstrom/devices/light_gaetan/brightness/command" ]
    }
    
    // Example of blinds. Can be GR-KL200, GR-KL210, GR-KL220 (angle may not apply to all of them)
    Thing topic ds_bas_gaetan_blinds "Store Gaétan" {
    Channels:
        Type rollershutter : position     [ stateTopic="digitalstrom/devices/blinds_gaetan/shadePositionOutside/state", commandTopic="digitalstrom/devices/blinds_gaetan/shadePositionOutside/command" ]
        Type dimmer : angle     [ stateTopic="digitalstrom/devices/blinds_gaetan/shadeOpeningAngleOutside/state", commandTopic="digitalstrom/devices/blinds_gaetan/shadeOpeningAngleOutside/command" ]
    }
}
```

## Items configuration
*items/your_room.items*
```
// Energy counter (dSM12)
Number dss_circuit_chambres_energy { channel="mqtt:topic:mosquitto:ds_circuit_chambre:energyWs" }
Number dss_circuit_chambres_consumption { channel="mqtt:topic:mosquitto:ds_circuit_chambre:consumptionW" }

// Lights
Dimmer bas_gaetan_light "Lumière Gaetan" { channel="mqtt:topic:mosquitto:ds_bas_gaetan_light:state"}

// Blinds
Rollershutter bas_gaetan_blinds_position "Position Store Gaetan" { channel="mqtt:topic:mosquitto:ds_bas_gaetan_blinds:position"}
Dimmer bas_gaetan_blinds_angle "Angle Store Gaetan" { channel="mqtt:topic:mosquitto:ds_bas_gaetan_blinds:angle"}
```