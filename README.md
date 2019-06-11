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

## Run

You can either run the program directly or using Docker.

### Build and run

Run the following commands in order to build and run the program.

First of all, you need to have Go installed on your system. Actually, the code is tested using Go 1.7.3.
Set GOPATH variable pointing to the current folder of the project, build the program and then run it.
You need to launch the program with **sudo** because of the usage of the Bluetooth device.

```bash
export GOPATH=<path to project>

go install app

sudo bin/app
```

### Using Docker

```bash
docker build --rm -f "Dockerfile" -t gio-fog-node-go:latest .

docker run --rm -it --net host --privileged gio-fog-node-go:latest
```

## TODO
- Split monolithic file
- Provide REST interface to interact with currently connected devices
- Add configuration file for devices
- Add Dockerfile for different architectures
