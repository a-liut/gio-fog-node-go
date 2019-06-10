# gio-fog-node-go

Go implementation of the Giò system fog node.

It search for Gio-compliant devices and connectes to them providing a unified REST interface to let the rest of Gio system interact with devices.

To stop the program, send SIGINT signal. 

## Transport

The framework provided is able to support multiple transports to connect to devices.
Only the BLE transport is implemented in this moment.

In order to define a new transport, just implement the Start method and add it to the list of registered transports.
The framework will take care of its execution.

### BLE Transport

Transport implementation that allow the software to interact with BLE Gio-compliant devices.


#### BLEDevice 
BLEDevice is a representation for a device that is handled by the system.
The system is able to select the right interface and functions in order to handle several devices. Thus, specialization of this interface must be used in order to handle more devices.


## TODO
- Split monolithic file
- Provide REST interface to interact with currently connected devices
- Add configuration file for devices
