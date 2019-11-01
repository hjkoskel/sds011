/*
SDS011 simulator
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/hjkoskel/listserialports"
	"github.com/hjkoskel/sds011"
)

func main() {
	fmt.Printf("Single sensor SDS011 SIM")

	pSerialDevice := flag.String("s", "", "serial device file")
	pDeviceId := flag.String("id", "ABCD", "SDS011 ID in hex 16bit no 0xFFFF")
	pUiport := flag.Int("uiport", 8088, "Port for https hosting")
	pHttpsCrt := flag.String("crt", "./keys/https-server.crt", "crt file for local ui")
	pHttpsKey := flag.String("key", "./keys/https-server.key", "key file for local ui")

	flag.Parse()

	serialDeviceFileName := string(*pSerialDevice)
	if serialDeviceFileName == "" {
		fmt.Printf("Please define serial device. (-h for help)\nList of serial ports\n")

		proped, errProbing := listserialports.ProbeSerialports()
		if errProbing != nil {
			fmt.Printf("Error probing serial port %v", errProbing.Error())
			os.Exit(-1)
		}
		for _, ser := range proped {
			fmt.Printf(ser.ToPrintoutFormat())
		}
		os.Exit(0)
	}

	sensorId, errIdparse := strconv.ParseInt(*pDeviceId, 16, 64)
	if errIdparse != nil {
		fmt.Printf("INVALID Device id %v  err=%v\n", *pDeviceId, errIdparse.Error())
		os.Exit(-1)
	}
	if (0xFFFF <= sensorId) || (sensorId < 0) {
		fmt.Printf("INVALID Device id %X\n", sensorId)
		os.Exit(-1)
	}

	//TODO load from disk if available
	fmt.Printf("\nThe sensor id is %X\n", sensorId)
	simsensor := InitSimSensor(uint16(sensorId))
	fmt.Printf("sim=%#v\n", simsensor)

	modelUpdates := make(chan SensorModel, 3)
	statusChanges := make(chan SensorModelStatus, 3) //Update UI

	packetsFromSensor := make(chan sds011.SDS011Packet, 6)
	packetsToSensor := make(chan sds011.SDS011Packet, 6)

	/*
		go func() {
			//TODO DEBUGGIKANAVALUUPPI
			for {
				fmt.Printf("-------------------------\n")
				fmt.Printf("model updates %v/%v\n", len(modelUpdates), cap(modelUpdates))
				fmt.Printf("status changes %v/%v\n", len(statusChanges), cap(statusChanges))
				fmt.Printf("packets from sensor %v/%v  to sensor %v/%v\n", len(packetsFromSensor), cap(packetsFromSensor), len(packetsToSensor), cap(packetsToSensor))
				fmt.Printf("-------------------------\n")
				time.Sleep(1500 * time.Millisecond)
			}
		}()
	*/

	serialLink, errLink := sds011.InitializeSerialLink(*pSerialDevice,
		packetsToSensor, sds011.SDS011TOSENSORSIZE,
		packetsFromSensor, sds011.SDS011FROMSENSORSIZE)
	if errLink != nil {
		fmt.Printf("SERIAL LINK FAIL %v\n", errLink.Error())
		return
	}

	//One sensor simple sim :)
	go func() { //HACK, single sensor
		for {
			simsensor.Model = <-modelUpdates
		}
	}()

	go func() {
		for {
			bytArr := <-simsensor.Output

			color.Set(color.FgCyan)
			fmt.Printf("to serial: %#X\n", bytArr)
			color.Unset()

			n, errWrite := serialLink.Serialport.Write(bytArr) //Should write all in one pass
			if errWrite != nil {
				fmt.Printf("Error writing %v\n", errWrite.Error())
			} else {
				if n != len(bytArr) {
					fmt.Printf("TODO HANDLE NON COMPLETE WRITE got %v wrote only %v", len(bytArr), n)
				}
			}
		}
	}()

	go func() { //allows to capture and print debug here :)
		for {
			msg := <-packetsToSensor //serialLink.Recieving
			color.Set(color.FgGreen)
			fmt.Printf("Recieved request %v (write to %v/%v)\n", msg.ToString(), len(simsensor.Input), cap(simsensor.Input))
			color.Unset()

			simsensor.Input <- msg
		}
	}()

	go func() {
		serialRunErr := serialLink.Run()
		if serialRunErr != nil {
			fmt.Printf("SERIAL LINK FAIL %v\n", serialRunErr.Error())
		} else {
			fmt.Printf("Serial link failed without any explanation\n")
		}
		os.Exit(-1)
	}()

	modelUpdateBySerial := make(chan SensorModel, 3)
	modelUpdateBySerial <- simsensor.Model
	go simsensor.Run(statusChanges, modelUpdateBySerial)

	errRun := runSingleSensorHttpsServer(modelUpdateBySerial, modelUpdates, statusChanges, *pUiport, *pHttpsCrt, *pHttpsKey)
	fmt.Printf("UI server failed %v\n", errRun.Error())

}
