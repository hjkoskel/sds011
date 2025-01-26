/*
Simple example how to use single SDS011 sensor on bus

And there is also debug option. Allows to listen all packets

On passive mode system program does not query device (like device ID, settings etc...)
*/

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"

	"github.com/fatih/color"
	"github.com/hjkoskel/listserialports"
	"github.com/hjkoskel/sds011"
	"github.com/pkg/term"
)

/*
func RunDebugMode(deviceFile string, devId uint16) error {
	//packetsFromSensor := make(chan sds011.SDS011Packet, 7)
	//packetsToSensor := make(chan sds011.SDS011Packet, 6)

	ser, err := sds011.CreateLinuxSerial(deviceFile)

	if err != nil {
		return err
	}
	go func() {
		for {
			pack := <-packetsFromSensor //ser.Recieving
			if pack.MatchToId(devId) {
				tNow := time.Now()
				fmt.Printf("%v %s\n", tNow, pack)
			}
		}
	}()

	return ser.Run()
}
*/

func getch() []byte {
	t, _ := term.Open("/dev/tty")
	term.RawMode(t)
	bytes := make([]byte, 3)
	numRead, err := t.Read(bytes)
	t.Restore()
	t.Close()
	if err != nil {
		return nil
	}
	return bytes[0:numRead]
}

func getOnlyAlphanum(in string) string {
	reg, _ := regexp.Compile("[^a-zA-Z0-9 ]+")
	return reg.ReplaceAllString(in, "")
}

func getIntegerUserInput(prompt string, base int, minvalue int, maxvalue int) (int, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	text, _ := reader.ReadString('\n')
	text = getOnlyAlphanum(text)

	per, perErr := strconv.ParseInt(text, base, 64)
	if perErr != nil {
		return 0, fmt.Errorf("invalid numerical input %v", perErr.Error())
	}

	if int(per) < minvalue || maxvalue < int(per) {
		return 0, fmt.Errorf("value %v out of range from %v -%v allowed", per, minvalue, maxvalue)
	}
	return int(per), nil
}

const (
	MEASUREDCOUNTERFILE     = "measuredcounter"
	MEASUREDCOUNTERFILE_TMP = "measuredcounter.tmp"
)

func loadMeasurementCounterFromFile() (int, error) {
	byt, errRead := os.ReadFile(MEASUREDCOUNTERFILE)
	if errRead != nil {
		return 0, fmt.Errorf("counter reading error %v", errRead.Error())
	}
	i, parseErr := strconv.ParseInt(string(byt), 10, 64)
	if parseErr != nil {
		return 0, fmt.Errorf("error parsing counter from file %v", parseErr.Error())
	}
	return int(i), nil
}

// Save with sync? bad?
func saveMeasuremntCounterToFile(counter int) error {
	f, err := os.Create(MEASUREDCOUNTERFILE_TMP)
	if err != nil {
		return err
	}
	defer f.Close()

	_, errW := io.WriteString(f, fmt.Sprintf("%v", counter)) //writes all.. so short
	if errW != nil {
		return errW
	}
	syncErr := f.Sync()
	if syncErr != nil {
		return syncErr
	}
	f.Close() //Do not care about error. If it is flushed

	return os.Rename(MEASUREDCOUNTERFILE_TMP, MEASUREDCOUNTERFILE)
}

func main() {
	pSerialDevice := flag.String("s", "", "serial device file")
	pPeriod := flag.Int("p", -1, "set period 0=30sec, 1= 1min 2=2min....")
	pDeviceId := flag.String("id", "FFFF", "device id in hex (filter)")
	//pDebug := flag.Bool("debug", false, "debug packages mode")
	pInteractive := flag.Bool("i", false, "interactive mode")
	/*
		pQueryMode := flag.Bool("q", false, "put query mode on (actively). Must do queries for getting data")
		pActiveMode := flag.Bool("a", false, "put active mode (activile) report actively by itself")
	*/

	theMeasCounter, errLoadCounter := loadMeasurementCounterFromFile()
	if errLoadCounter != nil {
		fmt.Printf("Error loading measurement counter %v, start from 0\n", errLoadCounter)
	} else {
		fmt.Printf("Starting from point count %v\n", theMeasCounter)
	}

	flag.Parse()

	serialDeviceFileName := string(*pSerialDevice)
	if serialDeviceFileName == "" {
		fmt.Printf("Please define serial device. (-h for help)\nList of serial ports\n")
		proped, _ := listserialports.Probe(false)
		for _, ser := range proped {
			fmt.Print(ser.ToPrintoutFormat())
		}
		os.Exit(0)
	}

	devId, errId := strconv.ParseInt(string(*pDeviceId), 16, 64)
	if errId != nil {
		fmt.Printf("Invalid Id, must be hex err=%v\n", errId.Error())
		return
	}

	if 0xFFFF < devId || devId < 0 {
		fmt.Printf("Device id %X is invalid\n", devId)
		return
	}

	/*if *pDebug {
		fmt.Printf("---- DEBUG MODE ----\n")
		//Debug mode just shows PACKETS
		err := RunDebugMode(serialDeviceFileName, uint16(devId))
		if err != nil {
			fmt.Printf("---FAILED %v---\n", err.Error())
		}
		return
	}*/

	if *pInteractive {
		err := interactiveMode(serialDeviceFileName, uint16(devId))
		if err != nil {
			fmt.Printf("ERR=%v\n", err.Error())
		}
	}

	//Now using only one sensor. It would be possible to put multiple sensors with different IDs on same bus (not tested yet)
	passive := int(*pPeriod) < 0

	serialLink, serialInitErr := sds011.CreateLinuxSerial(serialDeviceFileName)
	if serialInitErr != nil {
		fmt.Printf("Initializing serial port %v failed %v\n", serialDeviceFileName, serialInitErr.Error())
		return
	}
	fmt.Printf("Serial link is %#v\n", serialLink)

	/*
		go func() {
			linkError := serialLink.Run()
			if linkError != nil {
				fmt.Printf("Serial link fail %v\n", linkError)
				os.Exit(-1)
			}
			fmt.Printf("Serial link failed with no clear reason\n")
			os.Exit(-1)
		}()*/

	sensorResults := make(chan sds011.Result, 3)

	sensor := sds011.InitSds011(uint16(devId), passive, serialLink, sensorResults, theMeasCounter)
	if int(*pPeriod) < 0 {
		//No need to change
	} else {
		//Ok to fail at this point
		sensor.SetSettings(sds011.Sds011Settings{QueryMode: false, Period: byte(*pPeriod)}) //Active mode
	}

	go func() {
		for {
			errRun := sensor.Run()
			fmt.Printf("error run %s\n", errRun)
		}
	}()

	go func() {
		for {
			res := <-sensorResults
			color.Set(color.FgHiYellow)
			fmt.Printf("Sensor have result %v\n", res.ToString())
			color.Unset()
			countSaveErr := saveMeasuremntCounterToFile(res.MeasurementCounter)
			if countSaveErr != nil {
				color.Set(color.FgHiRed)
				fmt.Printf("counter save error %v\n", countSaveErr.Error())
				color.Unset()
			}
		}
	}()

	fmt.Printf("Sensor initial status is %#v\n", sensor)

	go func() {
		for {
			errSensor := <-sensor.ErrorsCh
			if errSensor != nil {
				color.Set(color.FgRed)
				fmt.Printf("\n\nSENSOR ERROR %v\n\n", errSensor.Error())
				color.Unset()
			}
		}
	}()

	go func() { //Demo how autodetect can be implemented (optional)
		for {
			otherId := <-sensor.DetectedSensor
			color.Set(color.FgMagenta)
			fmt.Printf("Other sensor id %X detected\n", otherId)
			color.Unset()
		}
	}()

	fmt.Printf("Going to run\n")

	for {
		runErr := sensor.Run()
		fmt.Printf("EXIT with %s\n", runErr)
	}
}
