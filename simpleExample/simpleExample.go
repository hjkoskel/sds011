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
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/hjkoskel/listserialports"
	"github.com/hjkoskel/sds011"
	"github.com/pkg/term"
)

func RunDebugMode(deviceFile string, devId uint16) error {
	packetsFromSensor := make(chan sds011.SDS011Packet, 7)
	packetsToSensor := make(chan sds011.SDS011Packet, 6)

	ser, err := sds011.InitializeSerialLink(
		deviceFile,
		packetsFromSensor, sds011.SDS011FROMSENSORSIZE,
		packetsToSensor, sds011.SDS011TOSENSORSIZE)
	if err != nil {
		return err
	}
	go func() {
		for {
			pack := <-packetsFromSensor //ser.Recieving
			if pack.MatchToId(devId) {
				tNow := time.Now()
				fmt.Printf("%v %v\n", tNow, pack.ToString())
			}
		}
	}()

	return ser.Run()
}

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
	fmt.Printf(prompt)
	text, _ := reader.ReadString('\n')
	text = getOnlyAlphanum(text)

	per, perErr := strconv.ParseInt(text, base, 64)
	if perErr != nil {
		return 0, fmt.Errorf("Invalid numerical input %v", perErr.Error())
	}

	if int(per) < minvalue || maxvalue < int(per) {
		return 0, fmt.Errorf("value %v out of range from %v -%v allowed", per, minvalue, maxvalue)
	}
	return int(per), nil
}

const (
	KEYCMD_QUERYMODE   = "q"
	KEYCMD_ACTIVEMODE  = "a"
	KEYCMD_WORKCOMMAND = "w"
)

func printInteractiveHelp() {
	fmt.Printf("---- Interactive commands ----\n")
	fmt.Printf("q = switch to query mode\n")
	fmt.Printf("a = switch to active mode\n")
	fmt.Printf("w = send work command\n")
	fmt.Printf("s = send stop command\n")
	fmt.Printf("z = read work status\n")
	fmt.Printf("d = query data\n")
	fmt.Printf("p = enter period setting\n")
	fmt.Printf("f = set id as filter\n")
	fmt.Printf("r = sync settings with sensor\n")
	fmt.Printf("t = status of settings now\n")
	fmt.Printf("g = report to lib that sensor should be off (use stop, not tested yet)")
	fmt.Printf("b = report to lib that sensor should be on (use stop, not tested yet)")
	fmt.Printf("h = print this help\n")

}

//This have colors :)
func interactiveMode(deviceFile string, devId uint16) error {
	printInteractiveHelp()

	packetsFromSensor := make(chan sds011.SDS011Packet, 7)
	packetsToSensor := make(chan sds011.SDS011Packet, 6)

	ser, err := sds011.InitializeSerialLink(
		deviceFile,
		packetsFromSensor, sds011.SDS011FROMSENSORSIZE,
		packetsToSensor, sds011.SDS011TOSENSORSIZE)
	if err != nil {
		return err
	}

	go func() {
		runErr := ser.Run()
		fmt.Printf("serial port run error %v\n", runErr.Error())
		os.Exit(-1)
	}()

	sensorResults := make(chan sds011.Sds011Result, 3)

	initialCounter, counterErr := loadMeasurementCounterFromFile()
	if counterErr != nil {
		if counterErr != nil {
			color.Set(color.FgRed)
			fmt.Printf("Counter error %v, starting from %v\n", counterErr.Error(), initialCounter)
			color.Unset()
		}
	} else {
		fmt.Printf("Initial counter value %v\n", initialCounter)
	}

	sensor := sds011.InitSds011(uint16(devId), false, packetsToSensor, packetsFromSensor, sensorResults, initialCounter)
	go sensor.Run()
	go func() {
		for {
			res := <-sensorResults
			color.Set(color.FgHiYellow)
			fmt.Printf("%v Sensor have result %v\n", res.ToString(), time.Now().String())
			color.Unset()
			countSaveErr := saveMeasuremntCounterToFile(res.MeasurementCounter)
			if countSaveErr != nil {
				color.Set(color.FgHiRed)
				fmt.Printf("counter save error %v\n", countSaveErr.Error())
				color.Unset()
			}

		}
	}()

	settings, errSettings := sensor.SyncSettings()
	if errSettings != nil {
		color.Set(color.FgRed)
		fmt.Printf("Error getting initial settings: %v\n", errSettings.Error())
		color.Unset()
	}

	for {
		arr := getch()
		c := string(arr[0])
		switch c {
		case "\x03":
			os.Exit(0)
			return nil //Hack exit
		case "b":
			fmt.Printf("Changing status so sensor should be off\n")
			sensor.PowerLine(false)
		case "g":
			fmt.Printf("Changing status so sensor should be on\n")
			sensor.PowerLine(true)

		case "q":
			fmt.Printf("switching to QUERY mode\n")
			settings.QueryMode = true
			sensor.SetSettings(settings) //Not capturing
			settings, errSettings = sensor.SyncSettings()
			if errSettings != nil {
				color.Set(color.FgRed)
				fmt.Printf("Error getting initial settings: %v\n", errSettings.Error())
				color.Unset()
			}
		case "a":
			fmt.Printf("switching to ACTIVE mode\n")
			settings.QueryMode = false
			sensor.SetSettings(settings) //Not capturing
			settings, errSettings = sensor.SyncSettings()
			if errSettings != nil {
				color.Set(color.FgRed)
				fmt.Printf("Error setting settings: %v\n", errSettings.Error())
				color.Unset()
			}
		case "w":
			fmt.Printf("going to work now\n")
			workErr := sensor.ChangeToWork(true)
			if workErr != nil {
				color.Set(color.FgRed)
				fmt.Printf("Error setting to work: %v\n", workErr.Error())
				color.Unset()
			}
		case "s":
			fmt.Printf("going to stop now\n")
			workErr := sensor.ChangeToWork(false)
			if workErr != nil {
				color.Set(color.FgRed)
				fmt.Printf("Error going to sleep: %v\n", workErr.Error())
				color.Unset()
			}
		case "z":
			fmt.Printf("query work status\n")
			working, workErr := sensor.IsWorking()
			if workErr != nil {
				color.Set(color.FgRed)
				fmt.Printf("Error setting to work: %v\n", workErr.Error())
				color.Unset()
			} else {
				if working {
					fmt.Printf("WORKING\n")
				} else {
					fmt.Printf("SLEEP\n")
				}
			}
		case "d":
			fmt.Printf("sending query data\n")
			workErr := sensor.DoQuery()
			if workErr != nil {
				color.Set(color.FgRed)
				fmt.Printf("Error sending query: %v\n", workErr.Error())
				color.Unset()
			}
		case "p":
			per, perErr := getIntegerUserInput("Enter period 0-30 minute:", 10, 0, 30)
			if perErr != nil {
				color.Set(color.FgRed)
				fmt.Printf("\n%v\n", perErr.Error())
				color.Unset()
			} else {

				settings.Period = byte(per)
				sensor.SetSettings(settings) //Not capturing
				settings, errSettings = sensor.SyncSettings()
				if errSettings != nil {
					color.Set(color.FgRed)
					fmt.Printf("Error setting settings: %v\n", errSettings.Error())
					color.Unset()
				}
			}
		case "f":
			fil, filErr := getIntegerUserInput("give new for ID filter in hex", 16, 0, 0xFFFF)
			if filErr != nil {
				color.Set(color.FgRed)
				fmt.Printf("Error setting settings: %v\n", errSettings.Error())
				color.Unset()
			} else {
				sensor.Id = uint16(fil)
			}
		case "r":
			fmt.Printf("syncing sensor settings..\n")
			newSet, errSet := sensor.SyncSettings()
			if errSet != nil {
				color.Set(color.FgRed)
				fmt.Printf("Error syncing settings: %v\n", errSet.Error())
				color.Unset()
			} else {
				settings = newSet
				fmt.Printf("Synced settings %#v\n", settings)
			}
		case "t":
			fmt.Printf("Settings on computer %#v\n", settings)
		case "h":
			printInteractiveHelp()
		}
	}
}

const (
	MEASUREDCOUNTERFILE     = "measuredcounter"
	MEASUREDCOUNTERFILE_TMP = "measuredcounter.tmp"
)

func loadMeasurementCounterFromFile() (int, error) {
	byt, errRead := ioutil.ReadFile(MEASUREDCOUNTERFILE)
	if errRead != nil {
		return 0, fmt.Errorf("Counter reading error %v", errRead.Error())
	}
	i, parseErr := strconv.ParseInt(string(byt), 10, 64)
	if parseErr != nil {
		return 0, fmt.Errorf("Error parsing counter from file %v", parseErr.Error())
	}
	return int(i), nil
}

//Save with sync? bad?
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
	pDebug := flag.Bool("debug", false, "debug packages mode")
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
		proped, _ := listserialports.ProbeSerialports()
		for _, ser := range proped {
			fmt.Printf(ser.ToPrintoutFormat())
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

	if *pDebug {
		fmt.Printf("---- DEBUG MODE ----\n")
		//Debug mode just shows PACKETS
		err := RunDebugMode(serialDeviceFileName, uint16(devId))
		if err != nil {
			fmt.Printf("---FAILED %v---\n", err.Error())
		}
		return
	}

	if *pInteractive {
		err := interactiveMode(serialDeviceFileName, uint16(devId))
		if err != nil {
			fmt.Printf("ERR=%v\n", err.Error())
		}
	}

	//Now using only one sensor. It would be possible to put multiple sensors with different IDs on same bus (not tested yet)
	passive := int(*pPeriod) < 0

	packetsFromSensor := make(chan sds011.SDS011Packet, 6)
	packetsToSensor := make(chan sds011.SDS011Packet, 6)

	serialLink, serialInitErr := sds011.InitializeSerialLink(
		serialDeviceFileName,
		packetsFromSensor, sds011.SDS011FROMSENSORSIZE,
		packetsToSensor, sds011.SDS011TOSENSORSIZE)

	if serialInitErr != nil {
		fmt.Printf("Initializing serial port %v failed %v\n", serialDeviceFileName, serialInitErr.Error())
		return
	}
	fmt.Printf("Serial link is %#v\n", serialLink)
	go func() {
		linkError := serialLink.Run()
		if linkError != nil {
			fmt.Printf("Serial link fail %v\n", linkError)
			os.Exit(-1)
		}
		fmt.Printf("Serial link failed with no clear reason\n")
		os.Exit(-1)
	}()

	sensorResults := make(chan sds011.Sds011Result, 3)

	sensor := sds011.InitSds011(uint16(devId), passive, packetsToSensor, packetsFromSensor, sensorResults, theMeasCounter)
	if int(*pPeriod) < 0 {
		//No need to change
	} else {
		//Ok to fail at this point
		sensor.SetSettings(sds011.Sds011Settings{QueryMode: false, Period: byte(*pPeriod)}) //Active mode
	}

	go sensor.Run()

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
	sensor.Run()
}
