package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

//var modelledSensors []SimSensor

// Single sensor server
func runSingleSensorHttpsServer(
	modelUpdatedBySerial chan SensorModel, modelUpdatedByUser chan SensorModel, simStatusUpdating chan SensorModelStatus,
	uiport int, httpsCrt string, httpsKey string) error {
	modelNow := <-modelUpdatedBySerial //Initial is needed for reason that channel direction must not change

	go func() {
		for {
			modelNow = <-modelUpdatedBySerial
			fmt.Printf("TODO MODEL UPDATED TO %#v\n", modelNow)
		}
	}()

	fmt.Printf("\n\nStarting server with %#v\n", modelNow)

	sensorStatusNow := SensorModelStatus{}
	go func() {
		for {
			sensorStatusNow = <-simStatusUpdating // simStatusUpdating
		}
	}()

	fs := http.FileServer(http.Dir("simui"))

	r := mux.NewRouter()

	r.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		//fmt.Printf("TODO HANDLE STATUS QUERY")
		b, _ := json.Marshal(sensorStatusNow)
		w.Write(b)
	})

	r.HandleFunc("/model", func(w http.ResponseWriter, r *http.Request) {
		//fmt.Printf("HANDLING QUERY %v\n", r.Method)
		if r.Method == "POST" {
			postbody, errRead := io.ReadAll(r.Body)
			if errRead != nil {
				w.Write([]byte(fmt.Sprintf("Reading POST request failed %v", errRead.Error())))
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			mod := SensorModel{}
			errMarsh := json.Unmarshal(postbody, &mod)
			if errMarsh != nil {
				w.Write([]byte(fmt.Sprintf("Invalid payload %v", errMarsh.Error())))
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			fmt.Printf("updating model to %#v\n", mod)
			if len(modelUpdatedByUser) < cap(modelUpdatedByUser) { //TODO VALIDITY CHECK HERE.. RETURN ERROR IF ID is in use
				modelUpdatedByUser <- mod
				modelNow = mod
			} else {
				fmt.Printf("MODEL UPDATE KANAVA TUKOSSA\n")
			}
		}

		//Report response
		b, _ := json.Marshal(modelNow)
		w.Write(b)
	})

	r.PathPrefix("/").Handler(fs)
	fmt.Printf("\n\nServing local UI on port %v\n", uiport)
	return http.ListenAndServeTLS(fmt.Sprintf(":%v", uiport), httpsCrt, httpsKey, r)
}
