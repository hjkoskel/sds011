# sds011
SDS011 air quality sensor interface for golang

Design goal is to also support multiple SDS011 sensors on same bus (using uart<->RS485 etc..)
At the moment I own only one sensor. And do not have real need for multi sensor support. And multisensor is not tested

# Structure

The lowest level is serial link library that convert serial port resource into
channels.

There is *Conn interface* for that. 
~~~go
type Conn interface {
	Send(packet Packet) error
	Recieve() (*Packet, error) //Return immediately. nil if not yet complete packet
	Close() error
}
~~~

The *LinuxConn* is one implementations for linux targets.

Over that there is sensor layer that keeps sensor model state and converts packages to measurement results
Sensor layer reacts only packages that have specified device id

## Serial link layer
Initialize link with. function
~~~go
func CreateLinuxSerial(deviceportName string) (*LinuxConn, error)
~~~

Then call Recieve functions of Conn when need to recieve or send.

Or use Sensor layer sds011 for handling messaging


## Sensor layer

Sensor layer is model of sensor and filtering mechanism

~~~go
func InitSds011(Id uint16, passive bool, conn Conn, resultCh chan Result, initialMeasurementCounter int) Sds011 {
 ~~~ 


When calling .Run method for this sensor, sensor runs and results coming from serial interface are processed result to channel


# Simulator
This package includes also crude sds011 sensor simulator program.
It hosts its own user interface for simulated sensor.

Simulator allows to simulate also error modes. Like disconnecting RX wire
or bad RS485 driver (echo 0 when direction changes etc...). Or bytes from other communication protocols moving in wires while there is no need for sds011 (bad design)

For using simulator on pc without sensor, you need real rs232 loopback cables from port to port or use two usb-ttl cables in usb-ttl-ttl-usb config.
Or use **socat**

# TIP for socat

For testing with simulator, use following socat command

~~~sh
socat -d -d pty,raw,echo=0 pty,raw,echo=0
~~~

it will create pair of pty serial devices that are connected together. Open one with simulator and another with actual software