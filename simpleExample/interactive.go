package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/hjkoskel/sds011"
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

// This have colors :)
func interactiveMode(deviceFile string, devId uint16) error {
	printInteractiveHelp()

	ser, err := sds011.CreateLinuxSerial(deviceFile)
	if err != nil {
		return err
	}

	/*go func() {
		runErr := ser.Run()
		fmt.Printf("serial port run error %v\n", runErr.Error())
		os.Exit(-1)
	}()*/

	sensorResults := make(chan sds011.Result, 3)

	initialCounter, counterErr := loadMeasurementCounterFromFile()
	if counterErr != nil {

		color.Set(color.FgRed)
		fmt.Printf("Counter error %v, starting from %v\n", counterErr.Error(), initialCounter)
		color.Unset()

	} else {
		fmt.Printf("Initial counter value %v\n", initialCounter)
	}

	sensor := sds011.InitSds011(uint16(devId), false, ser, sensorResults, initialCounter)
	go func() {

		for {
			runErr := sensor.Run()
			color.Set(color.FgRed)
			fmt.Printf("sensor run err %s\n", runErr)
			color.Unset()
		}
	}()
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
			fmt.Printf("\n")
		}
	}()

	settings, errSettings := sensor.GetSettings()
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
			errSettings = sensor.SetSettings(settings) //Not capturing
			if errSettings != nil {
				color.Set(color.FgRed)
				fmt.Printf("Error getting initial settings: %v\n", errSettings.Error())
				color.Unset()
			}
		case "a":
			fmt.Printf("switching to ACTIVE mode\n")
			settings.QueryMode = false
			errSettings = sensor.SetSettings(settings) //Not capturing
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
				errSettings = sensor.SetSettings(settings) //Not capturing
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
			newSet, errSet := sensor.GetSettings()
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
