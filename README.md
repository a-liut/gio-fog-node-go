# gio-fog-node-go

Go implementation of the Giò Plant fog node.

It search for Giò-compliant devices and connects to them providing a unified REST interface to let the rest of Giò Plant platform interact with devices.
The connection is kept open until the program stops.

To stop the program, send SIGINT signal. 

## Transport

The framework implemented is able to support multiple transports to connect to devices.
Only the BLE transport is implemented in this moment.

In order to define a new transport, just implement the Start method and add it to the list of registered transports.
The framework will take care of its execution.

### BLE Transport

Transport implementation that allow the software to interact with BLE Gio-compliant devices.

#### BLEDevice 
BLEDevice is a representation for a device that is handled by the system.
The system is able to select the right interface and functions in order to handle several devices.
Thus, specialization of this interface must be used in order to handle more devices.

A BLEDevice stores a set of *Services* and *Characteristics* used to read published values produced by the connected device.
Furthermore, it specifies also *actions* that used to trigger defined behaviors of a device.

## Run

You can either run the program directly or using Docker.

### Build and run

fognode is developed as a Go module.
**sudo** is necessary due to the Bluetooth device usage

```bash
go build -o fognode cmd/fognode/main.go

./fognode
```

### Using Docker

```bash
docker build -t gio-fog-node-go:latest .

docker run -it --net host --privileged gio-fog-node-go:latest
```

## REST interface

The software exposes a REST API that allows clients to interact with connected devices getting data and triggerable actions.
The REST API is exposed by default on the port 5003. Used port can be overridden by setting GIO_FOG_NODE_SERVER_PORT environment variable.

- GET /devices: fetch all connected devices

    Example response:
    
    ```json
    [
      {
        "id": "FE:F4:1C:74:66:B3",
        "name": "BBC micro:bit [zotut]",
        "characteristics": [
          {
            "uuid": "02759250523e493b8f941765effa1b20",
            "name": "light"
          },
          {
            "uuid": "e95d9250251d470aa062fa1922dfa9a8",
            "name": "temperature"
          },
          {
            "uuid": "73cd7350d32c4345a543487435c70c48",
            "name": "moisture"
          },
          {
            "uuid": "ce9e7625c44341db9cb581e567f3ba93",
            "name": "watering"
          }
        ]
      }
    ]
    ```

- GET /devices/{deviceID}: get information about a single connected device

    Example response:
    
    ```json
    {
        "id": "FE:F4:1C:74:66:B3",
        "name": "BBC micro:bit [zotut]",
        "characteristics": [
          {
            "uuid": "02759250523e493b8f941765effa1b20",
            "name": "light"
          },
          {
            "uuid": "e95d9250251d470aa062fa1922dfa9a8",
            "name": "temperature"
          },
          {
            "uuid": "73cd7350d32c4345a543487435c70c48",
            "name": "moisture"
          },
          {
            "uuid": "ce9e7625c44341db9cb581e567f3ba93",
            "name": "watering"
          }
        ]
      }
    ```

- POST /devices/{deviceId}/act/{actuatorName}: trigger an action on the selected device

    Example response:
    
    - Successful response
        ```json
        {"message":"Done"}
        ```
    - Action not available
        ```json
        {"message":"action not recognized: asd"}
        ```

## TODO
- Modularize BLE devices
- Add configuration file for devices
- Add REST interface for remote configuration