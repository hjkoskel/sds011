# sds011
SDS011 air quality sensor interface for golang

Design goal is to also support multiple SDS011 sensors on same bus (using uart<->RS485 etc..)
At the moment I own only one sensor. And do not have real need for multi sensor support. And multisensor is not tested

# Structure

The lowest level is serial link library that convert serial port resource into
channels.

Over that there is sensor layer that keeps sensor model state and converts packages to measurement results
Sensor layer reacts only packages that have specified device id

## Serial link layer
Initialize link with. function

func InitializeSerialLink(
  deviceportName string,
  recievingCh chan SDS011Packet,
  expectedRecievePacketSize int,
  sendingCh chan SDS011Packet,
  expectedSendPacketSize int) (Sds011Serial, error) {

On client
  expectedRecievePacketSize is sds011.SDS011FROMSENSORSIZE
  sendingCh is sds011.SDS011TOSENSORSIZE

On simulator:
  expectedRecievePacketSize is sds011.SDS011TOSENSORSIZE
  expectedSendPacketSize is sds011.SDS011FROMSENSORSIZE  

Then call .Run() method and link runs

## Sensor layer

Sensor layer is model of sensor and filtering mechanism

When using single sensor, just pass channels to sensor layer

For multiple sensors, extra goroutine is needed for distributing
data coming from serial link layer to each sensor.

create sensor model by calling
func InitSds011(
  Id uint16,
  passive bool,
  toSensorCh chan SDS011Packet,
  fromSensorCh chan SDS011Packet,
  resultCh chan Sds011Result,
  initialMeasurementCounter int) Sds011 {

on passive mode, software is not sending queries for sensor.

When calling .Run method for this sensor, sensor runs and results are coming to resultCh
Initial counter is needed for tracking up how many measurements (approx) are made. This will give rough estimate when sensor replacement is needed (laser/detector weardown in few years).  Library user must plan and implement how to persist counter

If some setting have to be changed. Please check example

func interactiveMode(deviceFile string, devId uint16) error {

from "simple example". And google sds011 datasheet for further technical details

# Simulator
This package includes also crude sds011 sensor simulator program. Simulate sensor without filling that with dust.
It hosts its own user interface for simulated sensor. Simulator allows to simulate also error modes. Like disconnecting RX wire
or bad RS485 driver (echo 0 when direction changes etc...). Or bytes from other communication protocols moving in wires while there is no need for sds011 (bad design)

For using simulator on pc without sensor, you need real rs232 loopback cables from port to port or use two usb-ttl cables in usb-ttl-ttl-usb config.
