/*
Sensor model

Sensor model state is loaded from disk and manipulated by user while running
This models single sensor

It is important to notice that simulated sensor recieves all messages "ok".
It just acts as faulty sensor (or comm link) when needed.

*/

package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/hjkoskel/sds011"
)

type SimSensor struct {
	Input             chan sds011.SDS011Packet
	outputQueue       chan sds011.SDS011Packet //allows to do all kind of crazy things
	Output            chan []byte              //Writes out burst of bytes
	Model             SensorModel              //This is loaded, changed...stored etc..
	SensorModelStatus SensorModelStatus
}

func InitSimSensor(id uint16) SimSensor {
	result := SimSensor{
		Input:       make(chan sds011.SDS011Packet, 10),
		outputQueue: make(chan sds011.SDS011Packet, 10),
		Output:      make(chan []byte, 10),
	}
	result.Model = SensorModel{Connectivity: ConnectivityModel{RxConnected: true, TxConnected: true}, PowerOn: true}
	result.Model.SensorMem = SensorMemory{Id: id, VersionYear: 19, VersionMonth: 9, VersionDay: 28, Period: 0, QueryMode: false}
	return result
}

//Separate settings and status
type SensorModelStatus struct {
	Working            bool   `json:"working"`            //- working or sleep (depend on period and last time)
	MeasurementCounter int    `json:"measurementCounter"` //- measurement counter (for sim)
	RxPacketCounter    int    `json:"rxPacketCounter"`    //- packet counters
	TxPacketCounter    int    `json:"txPacketCounter"`    //- packet counters
	SmallRegNow        uint16 `json:"smallRegNow"`        //Update these. Report by time or by clock
	LargeRegNow        uint16 `json:"largeRegNow"`
	BurnEventCounter   int    `json:"burnEventCounter"` //How many persistent save events happened
}

type SensorModel struct {
	SensorMem      SensorMemory      `json:"sensorMem"`
	PowerOn        bool              `json:"powerOn"` //- is powered up (toggling this allows to do "power reset")
	SmallParticles SignalModel       `json:"smallParticles"`
	LargeParticles SignalModel       `json:"largeParticles"`
	Connectivity   ConnectivityModel `json:"connectivity"` //Allow simulate communication conditions
}

type ConnectivityModel struct {
	RxConnected         bool `json:"rxConnected"`         //- rx line connected ( computer-> sensor)  RX not connected. Sensor do not react :D
	TxConnected         bool `json:"txConnected"`         //- tx line connected (sensor -> computer).
	ShortCircuit        bool `json:"shortCircuit"`        //rx and tx lines are connected together ERROR MODE
	DirectionChangeNull bool `json:"directionChangeNull"` //RS485 artefact. when recieve/transit changes there is null character
	IncompletePackages  bool `json:"incompletePackages"`  //Not all bytes are coming
	InvalidCRC          bool `json:"invalidCRC"`          //Wrong CRC, easy test
	IdleCharacters      bool `json:"idleCharacters"`      //Random line noise in between packets
}

type SignalModel struct { //Works as floats.. registers report as 10*
	Noise     float64 `json:"noise"` //in range [value-noise, value+noise]
	Offset    float64 `json:"offset"`
	Period    int64   `json:"period"`    //In milliseconds, sine period
	Phase     int64   `json:"phase"`     //In milliseconds.
	Amplitude float64 `json:"amplitude"` // offset-amplitude to offset+amplitude
}

type SensorMemory struct {
	Id           uint16 `json:"id"`
	VersionYear  byte   `json:"year"` // - Version:  year,month,day
	VersionMonth byte   `json:"month"`
	VersionDay   byte   `json:"day"`
	Period       byte   `json:"period"`
	QueryMode    bool   `json:"queryMode"`
}

//Can set any as long as its valid :)  TODO check duplicate Ids outside of this?
/*
TODO check later
func (p *SensorModel) Valid() bool {
	return p.SensorMem.Valid()
}

func (p *SensorMemory) Valid() bool {
	return (p.VersionDay < 32) && (p.VersionMonth < 13)
}
*/

func (p *SignalModel) Calc(t time.Time) float64 {
	ms := t.UnixNano() / (1000 * 1000)
	angle := 2.0 * math.Pi * math.Mod(float64(ms+p.Phase), float64(p.Period))
	wave := math.Sin(angle) * p.Amplitude
	if p.Period == 0 {
		wave = 0
	}
	return math.Max(0, rand.Float64()*(p.Noise*2.0-1.0)+wave+p.Offset)
}

func (p *SimSensor) reactToPackage(pack sds011.SDS011Packet, sensorUpdating chan SensorModel) (sds011.SDS011Packet, error) {
	if !pack.Valid { //Maybe this is tested in somewhere else beforehand
		return sds011.SDS011Packet{}, fmt.Errorf("INVALID PACKET")
	}
	if pack.CommandID != sds011.COMMANDID_CMD {
		return sds011.SDS011Packet{}, fmt.Errorf("Simulator understands only commandId=0xB4")
	}

	write := pack.GetIsWrite()
	switch pack.Data[0] {
	case sds011.FUNNUMBER_REPORTINGMODE:
		if write {
			p.Model.SensorMem.QueryMode, _ = pack.GetQueryMode()
			p.SensorModelStatus.BurnEventCounter++ //Important to count memory wear out
			sensorUpdating <- p.Model
		}
		return sds011.NewPacket_SetQueryModeReply(p.Model.SensorMem.Id, write, p.Model.SensorMem.QueryMode), nil
	case sds011.FUNNUMBER_QUERYDATA:
		return sds011.NewPacket_DataReply(p.Model.SensorMem.Id, p.SensorModelStatus.SmallRegNow, p.SensorModelStatus.LargeRegNow), nil
	case sds011.FUNNUMBER_SETID:
		if write {
			id, idErr := pack.GetSetId()
			if idErr != nil {
				return sds011.SDS011Packet{}, idErr
			}
			p.Model.SensorMem.Id = id
			p.SensorModelStatus.BurnEventCounter++ //Important to count memory wear out
			sensorUpdating <- p.Model
		}
		return sds011.NewPacket_SetIdReply(p.Model.SensorMem.Id), nil
	case sds011.FUNNUMBER_SLEEPWORK:
		//TODO RESETOI ODOTTELU JOS VAIHTAA TILAA
		if write {
			p.SensorModelStatus.Working, _ = pack.GetWorkMode()
			fmt.Printf("WRITING WORK MODE SETTING TO %v\n", p.SensorModelStatus.Working)
		}
		//fmt.Printf("\nSim working=%v\n\n", p.SensorModelStatus.Working)
		return sds011.NewPacket_SetWorkModeReply(p.Model.SensorMem.Id, write, p.SensorModelStatus.Working), nil
	case sds011.FUNNUMBER_PERIOD:
		if write {
			per, errPeriod := pack.GetPeriod() //TODO limit check
			if errPeriod != nil {
				return sds011.SDS011Packet{}, errPeriod
			}
			p.Model.SensorMem.Period = per
			p.SensorModelStatus.BurnEventCounter++ //Important to count memory wear out
			sensorUpdating <- p.Model
		}
		return sds011.NewPacket_SetPeriodReply(p.Model.SensorMem.Id, write, p.Model.SensorMem.Period), nil
	case sds011.FUNNUMBER_VERSION:
		return sds011.NewPacket_QueryVersionReply(p.Model.SensorMem.Id, p.Model.SensorMem.VersionYear, p.Model.SensorMem.VersionMonth, p.Model.SensorMem.VersionDay), nil
	}

	return sds011.SDS011Packet{}, fmt.Errorf("Invalid function %v", pack.Data[0])
}

const (
	INTERVALIDLECHARS = 1500
)

//Sends package based on ConnectivityModel
func (p *SimSensor) sendRoutine() {
	lastTrashTime := time.Now()
	for {
		if 0 < len(p.outputQueue) {
			p.Output <- p.Model.Connectivity.TrashSignal(<-p.outputQueue)
			p.SensorModelStatus.TxPacketCounter++
		}
		if p.Model.Connectivity.IdleCharacters {
			if (time.Millisecond * INTERVALIDLECHARS) < time.Since(lastTrashTime) {
				junk := make([]byte, 9)
				for i := range junk {
					junk[i] = byte(rand.Uint32() & 0xFF)
				}
				p.Output <- junk //[]byte{0, 2, 1, 4, 6} //Todo random mess
				lastTrashTime = time.Now()
			}
		}
		time.Sleep(50 * time.Millisecond) //Give process time
	}
}

//Trash signal only if needed
func (p *ConnectivityModel) TrashSignal(pack sds011.SDS011Packet) []byte {
	arr := pack.ToBytes()
	if p.InvalidCRC {
		arr[len(arr)-2] += 1
	}

	if p.DirectionChangeNull {
		arr = append([]byte{0}, arr...)
		arr = append(arr, 0)
	}
	if p.IncompletePackages { //Cut away from end reciever might keep waiting?
		arr = arr[0 : len(arr)-4]
	}
	return arr
}

func (p *SimSensor) Run(statusUpdatingCh chan SensorModelStatus, sensorUpdating chan SensorModel) {
	//Routine for timing sequencing
	go p.sendRoutine()
	go func() {
		fmt.Printf("\nSTARTING SENSOR TIMING ROUTINE\n")
		prevMeasCompleteTime := time.Unix(0, 0)
		for {

			since := time.Since(prevMeasCompleteTime).Seconds()
			per := float64(p.Model.SensorMem.Period) * 60
			if per == 0 {
				per = 30
			}
			p.SensorModelStatus.Working = (per - since) <= 30 //30sec before result put fan on
			//fmt.Printf("since=%v period=%vsec working=%v\n", since, per, p.SensorModelStatus.Working)
			if per < since {
				tNow := time.Now()
				smallResult := p.Model.SmallParticles.Calc(tNow)
				largeResult := p.Model.LargeParticles.Calc(tNow)
				fmt.Printf("Modelling small=%v large=%v\n", smallResult, largeResult)
				p.SensorModelStatus.SmallRegNow = uint16(smallResult * 10)
				p.SensorModelStatus.LargeRegNow = uint16(largeResult * 10)
				p.SensorModelStatus.MeasurementCounter++
				prevMeasCompleteTime = time.Now()

				if !p.Model.SensorMem.QueryMode {
					p.outputQueue <- sds011.NewPacket_DataReply(p.Model.SensorMem.Id, p.SensorModelStatus.SmallRegNow, p.SensorModelStatus.LargeRegNow)
				}
				if per < 30 {
					fmt.Printf("Measurement done, shutting down\n")
					p.SensorModelStatus.Working = false
				}
			}
			if len(statusUpdatingCh) < cap(statusUpdatingCh) {
				statusUpdatingCh <- p.SensorModelStatus
			}

			time.Sleep(500 * time.Millisecond)
		}
	}()

	//Reacts to input
	for {
		inp := <-p.Input
		p.SensorModelStatus.RxPacketCounter++
		fmt.Printf("Processing input with connectivity %#v\n", p.Model.Connectivity)
		if p.Model.Connectivity.RxConnected {
			if p.Model.Connectivity.ShortCircuit {
				p.Output <- inp.ToBytes() //Immediately report same back as fast wires would :D
			} else {
				if inp.MatchToId(p.Model.SensorMem.Id) {
					respPack, respErr := p.reactToPackage(inp, sensorUpdating)
					if respErr != nil {
						fmt.Printf("ERROR %v\n\n", respErr.Error())
					} else {
						fmt.Printf("\nGiving response %v\n", respPack.ToString())
						trashed := p.Model.Connectivity.TrashSignal(respPack)
						fmt.Printf("Trashed=%X (%v bytes) to out %v/%v\n", trashed, len(trashed), len(p.Output), cap(p.Output))
						p.Output <- trashed
						p.SensorModelStatus.TxPacketCounter++
					}
				} else {
					fmt.Printf("Packet ID=%v,  no match to simulator %v\n", inp.DeviceID, p.Model.SensorMem.Id)
				}
			}
		}

	}
}
