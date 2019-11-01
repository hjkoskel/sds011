# Simulator for SDS011

This program is for simulating single SDS011 sensor.  Case where there are multiple sensors on same serial port (via RS485 for example) can be simulated with minor modifications

I am able to implement multisensor support if someone really needs that :)


# Usage

Simulator uses web browser interface hosted by simulator

Simulator hosts pages using https on localhost. Https is needed because some browser features are available only on https.

Software can be compiled as normal golang program by go build
Before using, some keys have to be generated. From sds011sim working dir

```
mkdir keys
cd keys

openssl genrsa -out https-server.key 2048
openssl ecparam -genkey -name secp384r1 -out https-server.key
openssl req -new -x509 -sha256 -key https-server.key -out https-server.crt -days 3650
```

then program ui is available at https://127.0.0.1:8088 when simulator runs

Help command line switch
```
./sds011sim -h
```
Tells more about program usage.
