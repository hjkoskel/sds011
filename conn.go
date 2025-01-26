/*
SDS011conn
is connection for writing and reading packages from serial line

This is interface. Different implementations are made for linux and tinygo-microcontroller environments
*/
package sds011

type Conn interface {
	Send(packet Packet) error
	Recieve() (*Packet, error) //Return immediately. nil if not yet complete packet
	Close() error
}
