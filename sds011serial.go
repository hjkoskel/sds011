/*
sds011serial

This module handles serial port link. Lowest possible
One instance per serial port. Possible to use on client or on simulator
Just transmits and recieves. Trying to prepare for case where there are multiple sensors on same bus

Ignores all invalid communication.

Following rules when capturing packages

- Packet starts ALWAYS with 0xAA Ends 0xAB
- PC -> sensor is 19 bytes ALWAYS
- sensor -> PC is 10

*/
package sds011

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/hjkoskel/listserialports"
	"golang.org/x/sys/unix"
)

type Sds011Serial struct {
	recieving    chan SDS011Packet
	transmitting chan SDS011Packet

	Serialport                *os.File // *serial.Port //Using this as public, simulator needs
	expectedRecievePacketSize int
	expectedSendPacketSize    int //Used for bug catching. Someone writes bad packet to wrong channel (same type for rx and tx)
}

func InitializeSerialLink(deviceportName string,
	recievingCh chan SDS011Packet, expectedRecievePacketSize int,
	sendingCh chan SDS011Packet, expectedSendPacketSize int) (Sds011Serial, error) {

	//Quick check. Linux only support. This SDS011 can not be used by multiple programs at same time

	result := Sds011Serial{
		expectedSendPacketSize:    expectedSendPacketSize,
		expectedRecievePacketSize: expectedRecievePacketSize,
		recieving:                 recievingCh,
		transmitting:              sendingCh,
	}

	portUsedByPids, _, errPortDetect := listserialports.FileIsInUse(deviceportName)
	if errPortDetect != nil {
		return result, fmt.Errorf("Serial port error %v", errPortDetect.Error())
	}
	if 0 < len(portUsedByPids) {
		return result, fmt.Errorf("Serial port %v is in use (by PID %#v)", deviceportName, portUsedByPids)
	}
	var errOpen error

	result.Serialport, errOpen = os.OpenFile(deviceportName, unix.O_RDWR|unix.O_NOCTTY|unix.O_NONBLOCK, 0666)
	if errOpen != nil {
		return result, fmt.Errorf("Serial device %v open error %v", deviceportName, errOpen.Error())
	}

	//No parity, one stop bit

	fd := result.Serialport.Fd()

	t := unix.Termios{
		Iflag:  unix.IGNPAR, //TODO break signal?
		Cflag:  unix.CREAD | unix.CLOCAL | unix.B9600 | unix.CS8,
		Ispeed: unix.B9600,
		Ospeed: unix.B9600,
	}
	t.Cc[unix.VMIN] = 0
	t.Cc[unix.VTIME] = 30 //Desiseconds

	if _, _, errno := unix.Syscall6(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(unix.TCSETS),
		uintptr(unsafe.Pointer(&t)),
		0,
		0,
		0,
	); errno != 0 {
		return result, fmt.Errorf("Syscall6 fail %v", errno.Error())
	}

	errNonBlock := unix.SetNonblock(int(fd), false)
	if errNonBlock != nil {
		return result, fmt.Errorf("Setting nonblock %v", errNonBlock.Error())
	}
	return result, nil
}

func getUptime() int64 {
	byt, err := ioutil.ReadFile("/proc/uptime")
	if err != nil {
		return 0 //Fallback
	}
	fie := strings.Fields(string(byt))
	fuptime, _ := strconv.ParseFloat(fie[0], 64)
	return int64(fuptime * 1000)
}

func CutToStartWith(firstMustBe byte, arr []byte) []byte {
	for i, b := range arr {
		if b == firstMustBe {
			return arr[i:]
		}
	}
	return []byte{} //Just line noise
}

/*
Depending on is this to sensor or from sensor
*/
func (p *Sds011Serial) Run() error {
	if p.Serialport == nil {
		return fmt.Errorf("Serial port not initialized")
	}
	respbuf := make([]byte, p.expectedRecievePacketSize)
	rxPack := SDS011Packet{}

	serReader := bufio.NewReader(p.Serialport)
	accumulated := []byte{}

	errHappended := make(chan error, 1)

	go func() {
		for {
			pkg := <-p.transmitting
			payload := pkg.ToBytes()
			if len(payload) != p.expectedSendPacketSize {
				errHappended <- fmt.Errorf("Invalide packet send size %v, expecting %v", len(payload), p.expectedSendPacketSize)
				return
			}

			n, wErr := p.Serialport.Write(payload)
			if wErr != nil {
				errHappended <- fmt.Errorf("Error writing to serial port %v\n", wErr.Error())
				return
			} else {
				if n != len(payload) {
					//Should not happen
					errHappended <- fmt.Errorf("Was not able to write package in one call %v out of %v", n, len(payload))
					break
				}
			}
		}
	}()

	for {
		if 0 < len(errHappended) {
			return <-errHappended
		}

		time.Sleep(100 * time.Millisecond) //More than 1 byte? using buffered anyway
		nRecieved, errRead := serReader.Read(respbuf)
		if errRead != nil {
			if errRead != io.EOF { //Something more bad than eof happend
				return fmt.Errorf("Error reading err=%v", errRead.Error())
			}
		}
		if 0 < nRecieved {
			accumulated = append(accumulated, respbuf[0:nRecieved]...)
			accumulated = CutToStartWith(SDS011PACKETSTART, accumulated)

			if p.expectedRecievePacketSize <= len(accumulated) {
				if accumulated[p.expectedRecievePacketSize-1] == SDS011PACKETSTOP {
					parseErr := rxPack.FromBytes(getUptime(), accumulated)
					if parseErr == nil {
						p.recieving <- rxPack
					} else {
						//fmt.Printf("INVALID PACKET PARSING %v\n", parseErr.Error())
					}
				} else {
					//fmt.Printf("Number of accumulated is %v, expected=%v end char=%X\n", len(accumulated), p.expectedRecievePacketSize, accumulated[p.expectedRecievePacketSize-1])
				}
				accumulated = []byte{} //Nothing recieved or done. Must be break?
			}
		} else {
			if 0 < len(accumulated) {
				accumulated = []byte{} //Nothing recieved. Must be break?
			}
		}
	}
}
