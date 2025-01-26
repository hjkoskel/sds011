//go:build !tinygo

package sds011

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"github.com/hjkoskel/listserialports"
	"golang.org/x/sys/unix"
)

type LinuxConn struct {
	f         *os.File
	serReader *bufio.Reader
	buf       []byte //keep old here
}

func (p *LinuxConn) Close() error {
	return p.f.Close()
}

func (p *LinuxConn) SendBytes(data []byte) error {
	_, err := p.f.Write(data)
	return err
}

func (p *LinuxConn) Send(packet Packet) error {
	_, err := p.f.Write(packet.ToBytes())
	return err
}

func (p *LinuxConn) Recieve() (*Packet, error) {
	respbuf := make([]byte, 1024)

	nRecieved, errRead := p.serReader.Read(respbuf)
	if errRead != nil {
		if errRead != io.EOF { //Something more bad than eof happend
			return nil, fmt.Errorf("error reading err=%v", errRead.Error())
		}
	}
	if nRecieved < 1 {
		return nil, nil
	}

	//Do we have enough bytes
	p.buf = append(p.buf, respbuf[0:nRecieved]...)
	p.buf = trimToPacketStart(p.buf)
	if !EnoughBytes(p.buf) {
		return nil, nil
	}

	rxPack := Packet{}
	parseErr := rxPack.FromBytes(GetUptime(), p.buf)
	p.buf = p.buf[1:] //Remove start index even ok or invalid packet for clearing up for next
	return &rxPack, parseErr
}

// Uses fixed settings for SDS0101
func CreateLinuxSerial(deviceportName string) (*LinuxConn, error) {

	//TESTTED  socat -d -d pty,raw,echo=0 pty,raw,echo=0
	if !strings.HasPrefix(deviceportName, "/dev/pts") { //Avoid issues with testing with socat
		portUsedByPids, _, errPortDetect := listserialports.FileIsInUseByPids(deviceportName)
		if errPortDetect != nil {
			return nil, fmt.Errorf("serial port error %v", errPortDetect.Error())
		}
		if 0 < len(portUsedByPids) {
			return nil, fmt.Errorf("serial port %v is in use (by PID %#v)", deviceportName, portUsedByPids)
		}
	}

	f, errOpen := os.OpenFile(deviceportName, unix.O_RDWR|unix.O_NOCTTY|unix.O_NONBLOCK, 0666)
	result := LinuxConn{
		f:         f,
		serReader: bufio.NewReader(f),
		buf:       []byte{},
	}
	if errOpen != nil {
		return &result, fmt.Errorf("serial device %v open error %v", deviceportName, errOpen.Error())
	}

	//No parity, one stop bit

	fd := result.f.Fd()

	t := unix.Termios{
		Iflag:  unix.IGNPAR, //TODO break signal?
		Cflag:  unix.CREAD | unix.CLOCAL | unix.B9600 | unix.CS8,
		Ispeed: unix.B9600,
		Ospeed: unix.B9600,
	}
	t.Cc[unix.VMIN] = 0
	t.Cc[unix.VTIME] = 30 //Desiseconds   TODO MAKE THIS WORKING AS TINYGO's version

	if _, _, errno := unix.Syscall6(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(unix.TCSETS),
		uintptr(unsafe.Pointer(&t)),
		0,
		0,
		0,
	); errno != 0 {
		return &result, fmt.Errorf("syscall6 fail %v", errno.Error())
	}

	errNonBlock := unix.SetNonblock(int(fd), false)
	if errNonBlock != nil {
		return &result, fmt.Errorf("setting nonblock %v", errNonBlock.Error())
	}
	return &result, nil
}

// Util function if running on linux  TODO OTHER MODULE
func GetUptime() int64 {
	byt, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0 //Fallback
	}
	fie := strings.Fields(string(byt))
	fuptime, _ := strconv.ParseFloat(fie[0], 64)
	return int64(fuptime * 1000)
}
