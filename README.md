# gio-fog-node-go

The Gò Plants Fog Node searches for Giò-compliant devices and connects to them providing a unified REST interface to let the rest of Giò Plant platform interact with devices.
The connection is kept open until the program stops or the device disconnects.

To stop the program, send the SIGINT signal.

## How does it work

The tools start by scanning for BLE devices. When a new SmartVase is found, it connects to it and starts to receive data.
Data are then forwarded to registered callbacks if any. 

### Transport

The framework implemented is able to support multiple kinds of transport to connect to devices.
Only the BLE transport is implemented at this moment.

In order to define a new transport, just implement the Start method and add it to the list of registered transports.
The framework will take care of its execution.

#### BLE Transport

Transport implementation that allows the software to interact with BLE Gio-compliant devices.

It provides a notification mechanism that allows remote clients to be notified when a new reading is produced by a device.
The client register its *webhook* URL and when a new Reading is produced, Fog Node makes a POST HTTP call providing information about the device who produced the reading and the reading itself.

##### BLEDevice 
BLEDevice is a representation for a device that can be handled by the system.
The system is able to select the right interface and functions in order to handle several devices.
Thus, specialization of this interface must be used in order to handle more devices.

A BLEDevice stores a set of *Services* and *Characteristics* used to read published values produced by the connected device.
Furthermore, it specifies also *actions* that used to trigger defined behaviors of a device.
Actions names correspond to BLE Characteristics UUID.

## Run

You can either by building and running the program directly or by using Docker.

### Build and run

Fog Node is developed as a Go module.
WARNING: **sudo** is necessary due to the Bluetooth device usage.

```bash
go build -o fognode cmd/fognode/main.go

./fognode
```

### Using Docker

```bash
docker build -t gio-fog-node-go:latest .

docker run -it --net host --privileged gio-fog-node-go:latest
```

## REST API

The software exposes a REST API that allows clients to interact with connected devices getting data and available actions.
The REST API is exposed by default on the port 5003.
Port can be overridden by setting GIO_FOG_NODE_SERVER_PORT environment variable.

- POST /callbacks: creates a new callback for data notification.

    Example body:
  ```json
  {
  "url": "http://testurl:1234"
  }
  ```
  
  Example response:
  ```json
  {
  "code": 200,
  "message": "xxxxx"
  }
  ```
    
    
- DELETE /callbacks/{callbackUUID}: delete a callback given its UUID

  Example response:
  ```json
  {
  "code": 200,
  "message": "Done"
  }
  ```

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

- POST /devices/{deviceId}/actions/{actionName}: trigger an action on the selected device.
    It allows specifying a value to send to the device for the requested action

    Example body: .../actions/<characteristicUUID>
    ```json
    {
      "value": 42
    }
    ```

    Example response:
    
    - Successful response
      ```json
      {
        "code": 200,
        "message": "Done"
      }
      ```
    - Action not available
      ```json
      {
        "message":"action not recognized: test"
      }
      ```
      
## TODO
- Modularize BLE devices
- Add configuration file for devices
- Add REST interface for remote configuration
