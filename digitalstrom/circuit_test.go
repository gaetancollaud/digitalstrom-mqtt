package digitalstrom

import (
	"testing"
)

var circuitNormal, _ = getJson(`  {
                "name": "chambres",
                "dsid": "302ed89f43f00e40000123ac",
                "dSUID": "302ed89f43f0000000000e40000123ac00",
                "DisplayID": "000123ac",
                "hwVersion": 0,
                "hwVersionString": "12.1.4.0",
                "swVersion": "1.41.0.0 / DSP: 1.9.1.0",
                "armSwVersion": 19464192,
                "dspSwVersion": 17367296,
                "isUpToDate": true,
                "apiVersion": 772,
                "authorized": false,
                "hwName": "dSM12",
                "isPresent": true,
                "isValid": true,
                "busMemberType": 17,
                "hasDevices": true,
                "hasMetering": true,
                "hasBlinking": true,
                "VdcConfigURL": "",
                "VdcModelUID": "",
                "VdcHardwareGuid": "",
                "VdcHardwareModelGuid": "",
                "VdcImplementationId": "",
                "VdcVendorGuid": "",
                "VdcOemGuid": "",
                "ignoreActionsFromNewDevices": false
            }`)

var noDsid, _ = getJson(`  {
                "name": "chambres",
                "dsid": "",
                "dSUID": "302ed89f43f0000000000e40000123ac00",
                "DisplayID": "000123ac",
            }`)

var circuitManager = CircuitsManager{}

func TestSupportedCircuit(t *testing.T) {
	expectBool(t, circuitManager.supportedCircuit(circuitNormal), true, "heater should be supported")
	expectBool(t, circuitManager.supportedCircuit(noDsid), false, "noDsid should not be supported")
}
