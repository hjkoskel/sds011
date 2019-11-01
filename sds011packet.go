/*
For unpacking and packing SDS011 message packets
*/

package sds011

import (
	"fmt"
	"math"
	"strings"
)

//Get optimized serial transmit by exact packet size
const (
	SDS011TOSENSORSIZE   = 19
	SDS011FROMSENSORSIZE = 10
)

const (
	SDS011PACKETSTART = 0xAA
	SDS011PACKETSTOP  = 0xAB
)

const (
	COMMANDID_CMD       = 0xB4
	COMMANDID_RESPONSE  = 0xC5
	COMMANDID_DATAREPLY = 0xC0 //First byte in data is not function number
)

const ANYDEVICE = 0xFFFF
const ( //
	FUNNUMBER_REPORTINGMODE = 2 //NON- volatile
	FUNNUMBER_QUERYDATA     = 4
	FUNNUMBER_SETID         = 5 //NON-volatile
	FUNNUMBER_SLEEPWORK     = 6 //Is needed? Trigger?
	FUNNUMBER_PERIOD        = 8 //NON -volatile
	FUNNUMBER_VERSION       = 7
)

type SDS011Packet struct {
	CommandID byte
	DeviceID  uint16
	Checksum  byte
	Data      []byte
	Uptime    int64 //Extra information to carry around.. timestamping
	Valid     bool  //Is ok or not
}

func (p *SDS011Packet) MatchToId(id uint16) bool {
	return p.DeviceID == id || id == ANYDEVICE || (p.DeviceID == ANYDEVICE) //works for sim and client
}

func (p *SDS011Packet) ToString() string {
	if !p.Valid {
		return fmt.Sprintf("INVALID PACKET %X\n", p.ToBytes())
	}

	commandIdString := "UNKNOWN"
	switch p.CommandID {
	case COMMANDID_CMD:
		commandIdString = "fromPC"
	case COMMANDID_RESPONSE:
		commandIdString = "fromSensor"
	case COMMANDID_DATAREPLY:
		commandIdString = "data"
	}
	result := fmt.Sprintf("<SDS011:%X:%v ", p.DeviceID, commandIdString)
	if len(p.Data) == 0 {
		return result + "INVALIDNODATA>"
	}

	if p.CommandID == COMMANDID_DATAREPLY {
		res, errReg := p.GetMeasurement()
		//small, large, errReg := p.GetMeasurementSmallLargeRegs()
		if errReg != nil {
			result += fmt.Sprintf("INVALID %v", errReg.Error())
		} else {
			result += fmt.Sprintf("smallReg=%v largeReg=%v>", res.SmallReg, res.LargeReg)
		}
	} else {
		if p.GetIsWrite() {
			result += "w:"
		} else {
			result += "r:"
		}

		switch p.Data[0] {
		case FUNNUMBER_REPORTINGMODE:
			q, _ := p.GetQueryMode()
			if q {
				result += "mode:QUERY>"
			} else {
				result += "mode:ACTIVE>"
			}
		case FUNNUMBER_QUERYDATA:
			result += "QUERY>"
		case FUNNUMBER_SETID:
			id, _ := p.GetSetId()
			result += fmt.Sprintf("setId:%X>", id)
		case FUNNUMBER_SLEEPWORK:
			w, _ := p.GetWorkMode()
			if w {
				result += "WORK>"
			} else {
				result += "SLEEP>"
			}
		case FUNNUMBER_PERIOD:
			per, _ := p.GetPeriod()
			result += fmt.Sprintf("period=%v>", int(per))
		case FUNNUMBER_VERSION:
			ver, _ := p.GetVersionString()
			result += "version:" + ver + ">"
		default:
			result += fmt.Sprintf("INVALIDFUNCTION %v>", p.Data[0])
		}
	}
	return result
}

func (p *SDS011Packet) CalcChecksum() byte {
	result := byte(p.DeviceID & 0xFF)
	result += byte(p.DeviceID / 256)
	for _, b := range p.Data {
		result += b
	}
	return result
}

func (p *SDS011Packet) ChecksumOk() bool {
	return p.Checksum == p.CalcChecksum()
}

func (p *SDS011Packet) ToBytes() []byte {
	p.Checksum = p.CalcChecksum()
	result := []byte{SDS011PACKETSTART, p.CommandID}
	result = append(result, p.Data...)
	tail := []byte{byte(p.DeviceID / 256), byte(p.DeviceID & 0xFF), p.Checksum, SDS011PACKETSTOP}
	return append(result, tail...)
}

//trims line noise away
func trimToPacketStart(input []byte) []byte {
	result := []byte{}
	for iStart, v := range input {
		if v == SDS011PACKETSTART {
			result = input[iStart:]
		}
	}
	return result
}

//Require packet starting with 0xAA and end 0xAB  Remember to add uptime here. (exact timestamp)
func (p *SDS011Packet) FromBytes(uptimeNow int64, arr []byte) error {
	p.Uptime = uptimeNow
	p.Valid = false
	arr = trimToPacketStart(arr)

	if len(arr) < SDS011FROMSENSORSIZE {
		return fmt.Errorf("Invalid data size=%v at least %v requred", len(arr), SDS011FROMSENSORSIZE)
	}
	//Is larger packet? Check that first
	if SDS011FROMSENSORSIZE <= len(arr) {
		if arr[SDS011FROMSENSORSIZE-1] == SDS011PACKETSTOP {
			arr = arr[0:SDS011FROMSENSORSIZE]
		}
	}
	//usually transmitted packet have zeros.. so no danger that there is unintentional packet end
	if SDS011TOSENSORSIZE <= len(arr) {
		if arr[SDS011TOSENSORSIZE-1] == SDS011PACKETSTOP {
			arr = arr[0:SDS011TOSENSORSIZE]
		}
	}

	if (len(arr) != SDS011FROMSENSORSIZE) && (len(arr) != SDS011TOSENSORSIZE) {
		return fmt.Errorf("Invalid data size %v", len(arr))
	}
	if arr[0] != 0xAA {
		return fmt.Errorf("Invalid packet header %X", arr[0])
	}
	if arr[len(arr)-1] != SDS011PACKETSTOP {
		return fmt.Errorf("Invalid packet termination %X", arr[len(arr)-1])
	}
	p.CommandID = arr[1]

	if p.CommandID != COMMANDID_CMD && p.CommandID != COMMANDID_RESPONSE && p.CommandID != COMMANDID_DATAREPLY {
		return fmt.Errorf("Command ID 0x%X is not supported", p.CommandID)
	}

	p.Checksum = arr[len(arr)-2]
	p.DeviceID = uint16(arr[len(arr)-3]) + uint16(arr[len(arr)-4])<<8

	p.Data = arr[2 : len(arr)-4]

	switch p.CommandID {
	case COMMANDID_CMD:
		if len(arr) != 19 {
			return fmt.Errorf("Expect 19 long packet for commandID %v", COMMANDID_CMD)
		}

		switch p.Data[0] {
		case FUNNUMBER_REPORTINGMODE, FUNNUMBER_QUERYDATA, FUNNUMBER_SETID, FUNNUMBER_SLEEPWORK, FUNNUMBER_PERIOD, FUNNUMBER_VERSION:
			//OK
		default:
			return fmt.Errorf("Function %v not supportd with commandID 0x%X", p.Data[0], p.CommandID)
		}

	case COMMANDID_RESPONSE:
		if len(arr) != 10 {
			return fmt.Errorf("Expect 10 long packet for commandID %v", COMMANDID_RESPONSE)
		}
		//LACKS: FUNNUMBER_QUERYDATA
		switch p.Data[0] {
		case FUNNUMBER_REPORTINGMODE, FUNNUMBER_SETID, FUNNUMBER_SLEEPWORK, FUNNUMBER_PERIOD, FUNNUMBER_VERSION:
			//OK
		default:
			return fmt.Errorf("Function %v not supportd with commandID 0x%X", p.Data[0], p.CommandID)
		}

	case COMMANDID_DATAREPLY:
		if len(arr) != 10 {
			return fmt.Errorf("Expect 10 long packet for commandID %v", COMMANDID_DATAREPLY)
		}

	default:
		return fmt.Errorf("Invalid command id %v", p.CommandID)
	}

	if !p.ChecksumOk() {
		return fmt.Errorf("Checksum error")
	}
	p.Valid = true
	return nil
}

func (p *SDS011Packet) ToDebugText() string { //Like in manual
	raw := p.ToBytes()
	cmdIdLookup := map[byte]string{COMMANDID_CMD: "Query", COMMANDID_RESPONSE: "Response", COMMANDID_DATAREPLY: "datareply"}
	result := fmt.Sprintf("--- %v ", cmdIdLookup[p.CommandID]+" ---\n")
	for index, v := range raw {
		result += fmt.Sprintf("[%v]=%X\n", index, v)
	}
	return result + "-----------\n"
}

func boolToByte(boo bool) byte {
	if boo {
		return 1
	}
	return 0
}

/*
Functions for creating packages
*/
/*
Set data reporting mode
*/

func NewPacket_SetQueryMode(deviceId uint16, write bool, query bool) SDS011Packet {
	return SDS011Packet{
		CommandID: COMMANDID_CMD,
		DeviceID:  deviceId,
		Data:      []byte{FUNNUMBER_REPORTINGMODE, boolToByte(write), boolToByte(query), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Valid:     true,
	}
}

func NewPacket_SetQueryModeReply(deviceId uint16, write bool, query bool) SDS011Packet {
	return SDS011Packet{
		CommandID: COMMANDID_RESPONSE,
		DeviceID:  deviceId,
		Data:      []byte{FUNNUMBER_REPORTINGMODE, boolToByte(write), boolToByte(query), 0},
		Valid:     true,
	}
}

func (p *SDS011Packet) GetQueryMode() (bool, error) {
	if !p.Valid {
		return false, fmt.Errorf("Invalid packet")
	}
	if p.Data[0] != FUNNUMBER_REPORTINGMODE {
		return false, fmt.Errorf("Function number is not %v, it is %v", FUNNUMBER_REPORTINGMODE, p.Data[0])
	}
	return 0 < p.Data[2], nil
}

func (p *SDS011Packet) GetIsWrite() bool { //Some commands have write and read modes
	f := p.Data[0]
	if (f == FUNNUMBER_REPORTINGMODE) || (f == FUNNUMBER_SLEEPWORK) || (f == FUNNUMBER_PERIOD) {
		return 0 < p.Data[1]
	}
	return (f == FUNNUMBER_SETID) //Always writes if someone do not understand :D
}

/*
Query data command
*/

func NewPacket_QueryData(deviceId uint16) SDS011Packet {
	return SDS011Packet{
		CommandID: COMMANDID_CMD,
		DeviceID:  deviceId,
		Data:      []byte{FUNNUMBER_QUERYDATA, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Valid:     true,
	}
}

func NewPacket_DataReply(deviceId uint16, pm2_5 uint16, pm10 uint16) SDS011Packet {
	return SDS011Packet{
		CommandID: COMMANDID_DATAREPLY,
		DeviceID:  deviceId,
		Data:      []byte{byte(pm2_5 & 0xFF), byte(pm2_5 >> 8), byte(pm10 & 0xFF), byte(pm10 >> 8)},
		Valid:     true,
	}
}

type Sds011Result struct {
	MeasurementCounter int
	Uptime             int64
	SmallReg           uint16
	LargeReg           uint16
}

func millisecToString(ms int64) string {
	toks := []string{}
	total := ms
	if (1000 * 60 * 60) < ms {
		toks = append(toks, fmt.Sprintf("%vh", int64(math.Floor(float64(total)/(1000*60*60)))))
		total = total % (1000 * 60 * 60)
	}

	if (1000 * 60) < ms {
		toks = append(toks, fmt.Sprintf("%vmin", int64(math.Floor(float64(total)/(1000*60)))))
		total = total % (1000 * 60)
	}
	toks = append(toks, fmt.Sprintf("%vsec", int64(math.Floor(float64(total)/(1000)))))
	return strings.Join(toks, " ")
}

//NOTICE: non calibrated values, used for debug
func (p *Sds011Result) ToString() string {
	return fmt.Sprintf("count=%v %v PM2.5= %.1fµm/m³ PM10= %.1fµm/m³", p.MeasurementCounter, millisecToString(p.Uptime), float64(p.SmallReg)/10, float64(p.LargeReg)/10)
}

func (p *SDS011Packet) GetMeasurement() (Sds011Result, error) {
	if !p.Valid {
		return Sds011Result{}, fmt.Errorf("Invalid packet")
	}
	if p.CommandID != COMMANDID_DATAREPLY {
		return Sds011Result{}, fmt.Errorf("Not measurement packet commandid=%v", p.CommandID)
	}
	return Sds011Result{
		Uptime:   p.Uptime,
		SmallReg: uint16(p.Data[0]) + uint16(p.Data[1])*256,
		LargeReg: uint16(p.Data[2]) + uint16(p.Data[3])*256,
	}, nil

}

/*
Set device id.  DO NOT USE. Unless really wanted
*/
func NewPacket_SetId(deviceId uint16, newDeviceId uint16) SDS011Packet {
	return SDS011Packet{
		CommandID: COMMANDID_CMD,
		DeviceID:  deviceId,
		Data:      []byte{FUNNUMBER_SETID, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(newDeviceId >> 8), byte(newDeviceId & 0xFF)},
		Valid:     true,
	}
}
func NewPacket_SetIdReply(deviceId uint16) SDS011Packet {
	return SDS011Packet{
		CommandID: COMMANDID_RESPONSE,
		DeviceID:  deviceId,
		Data:      []byte{FUNNUMBER_SETID, 0, 0, 0},
		Valid:     true,
	}
}

func (p *SDS011Packet) GetSetId() (uint16, error) {
	if p.Data[0] != FUNNUMBER_SETID {
		return 0, fmt.Errorf("Function number is not SETID")
	}
	if len(p.Data) < 12 {
		return 0, fmt.Errorf("Invalid data length o SETID packet")
	}
	if p.CommandID == COMMANDID_CMD {
		return uint16(p.Data[11])<<8 + uint16(p.Data[12]), nil
	}
	if p.CommandID == COMMANDID_RESPONSE {
		return p.DeviceID, nil
	}
	return 0, fmt.Errorf("Invalid commandID %X", p.CommandID)
}

/*
4)Set device sleep work time
*/
func NewPacket_SetWorkMode(deviceId uint16, write bool, work bool) SDS011Packet {
	return SDS011Packet{
		CommandID: COMMANDID_CMD,
		DeviceID:  deviceId,
		Data:      []byte{FUNNUMBER_SLEEPWORK, boolToByte(write), boolToByte(work), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Valid:     true,
	}
}

func NewPacket_SetWorkModeReply(deviceId uint16, write bool, work bool) SDS011Packet {
	return SDS011Packet{
		CommandID: COMMANDID_RESPONSE,
		DeviceID:  deviceId,
		Data:      []byte{FUNNUMBER_SLEEPWORK, boolToByte(write), boolToByte(work), 0},
		Valid:     true,
	}
}

//Assuming valid packet (it have data, only valid function)
func (p *SDS011Packet) GetWorkMode() (bool, error) {
	if !p.Valid {
		return false, fmt.Errorf("Invalid packet")
	}
	/*
		if p.CommandID != COMMANDID_RESPONSE {
			return false, fmt.Errorf("Packet is not response")
		}
	*/
	if p.Data[0] != FUNNUMBER_SLEEPWORK {
		return false, fmt.Errorf("Function number is not %v, it is %v", FUNNUMBER_SLEEPWORK, p.Data[0])
	}
	return 0 < p.Data[2], nil
}

/*
Set working period
*/
func NewPacket_SetPeriod(deviceId uint16, write bool, period byte) SDS011Packet {
	return SDS011Packet{
		CommandID: COMMANDID_CMD,
		DeviceID:  deviceId,
		Data:      []byte{FUNNUMBER_PERIOD, boolToByte(write), period, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Valid:     true,
	}
}

func NewPacket_SetPeriodReply(deviceId uint16, write bool, period byte) SDS011Packet {
	return SDS011Packet{
		CommandID: COMMANDID_RESPONSE,
		DeviceID:  deviceId,
		Data:      []byte{FUNNUMBER_PERIOD, boolToByte(write), period, 0},
		Valid:     true,
	}
}

func (p *SDS011Packet) GetPeriod() (byte, error) {
	if !p.Valid {
		return 0, fmt.Errorf("Invalid packet")
	}
	/*
		if p.CommandID != COMMANDID_RESPONSE {
			return 0, fmt.Errorf("Packet is not response")
		}
	*/
	if p.Data[0] != FUNNUMBER_PERIOD {
		return 0, fmt.Errorf("Function number is not %v, it is %v", FUNNUMBER_PERIOD, p.Data[0])
	}
	return p.Data[2], nil
}

/*
Version
*/
func NewPacket_QueryVersion(deviceId uint16) SDS011Packet {
	return SDS011Packet{
		CommandID: COMMANDID_CMD,
		DeviceID:  deviceId,
		Data:      []byte{FUNNUMBER_VERSION, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Valid:     true,
	}
}

func NewPacket_QueryVersionReply(deviceId uint16, year byte, month byte, day byte) SDS011Packet {
	return SDS011Packet{
		CommandID: COMMANDID_RESPONSE,
		DeviceID:  deviceId,
		Data:      []byte{FUNNUMBER_VERSION, year, month, day},
		Valid:     true,
	}
}

func (p *SDS011Packet) GetVersionString() (string, error) {
	if !p.Valid {
		return "", fmt.Errorf("Invalid packet")
	}
	if p.Data[0] != FUNNUMBER_VERSION {
		return "", fmt.Errorf("Function number is not %v, it is %v", FUNNUMBER_VERSION, p.Data[0])
	}
	return fmt.Sprintf("%v.%v.%v", p.Data[1], p.Data[2], p.Data[3]), nil
}
