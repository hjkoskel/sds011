package sds011

import (
	"fmt"
	"math"
	"time"
)

const (
	TIMEOUTRESPONSE  = 500  //Query is sent, how long wait sensor response
	SYNCRETRYINTEVAL = 3000 //How long to wait before syncing settings from sensor
)

type Sds011 struct {
	PassiveMode    bool //Only listen
	SettingsInSync bool //If flips to offline (timeout etc... require settings check)

	Id              uint16         //Listen only these messages
	settings        Sds011Settings //Change thru method
	settingsChanged bool           //internal set true if settings were set even once.  false= just read now, change maybe later

	//Channels for reporting spontanious things
	resultCh       chan Sds011Result
	ErrorsCh       chan error  //Push nil if recovered or came online
	DetectedSensor chan uint16 //In case of multiple sensors. add item here when found valid package with new ID

	//Low level interface
	toSensor   chan SDS011Packet
	fromSensor chan SDS011Packet
	//Internal channels. Important to split inlet channels.. detecting reply is going to be much easier with this
	filtreplyFromSensor chan SDS011Packet

	tPrevResultTime time.Time //How long since data

	measurementCounter int //It is important to restore old readout and continue from there. Sensor wears down in each run

	//Power enable is not really needed. Sensor have stop command. But it is good to have switch as sensor reset
	powerEnable bool //If system have gpio controlled hiside switch for sensor. Stops counting time etc..
}

type Sds011Settings struct {
	QueryMode bool
	Period    byte   //0=30sec no sleep, 1=1 minute period
	Version   string //can be queried if active mode   READONLY
}

func (p *Sds011Settings) PeriodDuration() time.Duration {
	if p.Period == 0 {
		return time.Second * 30
	}
	return time.Minute * time.Duration(p.Period)
}

func InitSds011(Id uint16, passive bool, toSensorCh chan SDS011Packet, fromSensorCh chan SDS011Packet, resultCh chan Sds011Result, initialMeasurementCounter int) Sds011 {
	result := Sds011{
		PassiveMode:         passive,
		Id:                  Id,
		settings:            Sds011Settings{},
		toSensor:            toSensorCh,
		fromSensor:          fromSensorCh,
		filtreplyFromSensor: make(chan SDS011Packet, 1), //Just passing thru
		resultCh:            resultCh,
		ErrorsCh:            make(chan error, 2), //Optional... get error info from here
		DetectedSensor:      make(chan uint16, 10),
		measurementCounter:  initialMeasurementCounter, //What was counter when stopped (last reported)
		powerEnable:         true,
		tPrevResultTime:     time.Now(),
	}

	return result
}

//For reading and writing settings. Data query is different
func (p *Sds011) queryAndWaitResponse(query SDS011Packet) (SDS011Packet, error) {
	for 0 < len(p.filtreplyFromSensor) {
		<-p.filtreplyFromSensor //Clear up
	}
	for 0 < len(p.toSensor) {
		<-p.toSensor //Clear up
	}

	if !p.powerEnable {
		return SDS011Packet{}, fmt.Errorf("Power line not enabled") //Internal mess up if software makes queries while sensor is disabled
	}

	p.toSensor <- query
	tStart := time.Now()
	for time.Since(tStart) < time.Millisecond*TIMEOUTRESPONSE {
		if 0 < len(p.filtreplyFromSensor) {
			reply := <-p.filtreplyFromSensor
			if reply.CommandID == COMMANDID_RESPONSE { //Ignore other stuff. Like shorted rx tx echo back etc...
				return reply, nil
			}
		}
		time.Sleep(50 * time.Millisecond) //granularity
	}
	//TIMEOUT
	p.SettingsInSync = false
	return SDS011Packet{}, fmt.Errorf("timeout")
}

//If system have hiside power enable for sensor
func (p *Sds011) PowerLine(enabled bool) {
	if !p.powerEnable && enabled {
		//Switching on
		p.tPrevResultTime = time.Now() //Prevent counter "explosion"
	}
	p.powerEnable = enabled
}

/*
Settings
*/

func (p *Sds011) readQueryMode() (bool, error) {
	workModeReply, replyErr := p.queryAndWaitResponse(NewPacket_SetQueryMode(p.Id, false, false))
	if replyErr != nil {
		return false, replyErr
	}
	return workModeReply.GetQueryMode()
}

func (p *Sds011) writeQueryMode(queryMode bool) error {
	if p.PassiveMode {
		return fmt.Errorf("Write not allowed in passive mode")
	}

	workModeReply, replyErr := p.queryAndWaitResponse(NewPacket_SetQueryMode(p.Id, true, queryMode))
	if replyErr != nil {
		return replyErr
	}
	resp, err := workModeReply.GetQueryMode()
	if err != nil {
		return err
	}
	if resp != queryMode {
		return fmt.Errorf("Setting query mode to %v failed", queryMode)
	}
	return nil
}

func (p *Sds011) readPeriod() (byte, error) {
	periodReply, replyErr := p.queryAndWaitResponse(NewPacket_SetPeriod(p.Id, false, 0))
	if replyErr != nil {
		return 0, replyErr
	}
	return periodReply.GetPeriod()
}
func (p *Sds011) writePeriod(period byte) error {
	if p.PassiveMode {
		return fmt.Errorf("Write not allowed in passive mode")
	}

	periodReply, replyErr := p.queryAndWaitResponse(NewPacket_SetPeriod(p.Id, true, period))
	if replyErr != nil {
		return replyErr
	}
	resp, err := periodReply.GetPeriod()
	if err != nil {
		return err
	}
	if resp != period {
		return fmt.Errorf("setting period to %v failed. Reported %v", period, resp)
	}
	return nil
}

func (p *Sds011) readVersion() (string, error) {
	versionReply, replyErr := p.queryAndWaitResponse(NewPacket_QueryVersion(p.Id))
	if replyErr != nil {
		return "", replyErr
	}
	return versionReply.GetVersionString()
}

//Read what is going on device. Call if wanted
func (p *Sds011) readSettings() (Sds011Settings, error) {
	queryMode, queryModeErr := p.readQueryMode()
	if queryModeErr != nil {
		return p.settings, queryModeErr //Return something "neutral"
	}
	period, periodErr := p.readPeriod()
	if periodErr != nil {
		return p.settings, periodErr //Return something "neutral"
	}
	version, versionErr := p.readVersion()
	if versionErr != nil {
		return p.settings, versionErr //Return something "neutral"
	}
	return Sds011Settings{QueryMode: queryMode, Period: period, Version: version}, nil
}

//Response comes from result channel  TODO Timeout checking?
func (p *Sds011) DoQuery() error {
	pkg := NewPacket_QueryData(p.Id)
	p.toSensor <- pkg
	return nil
}

//IS NOT RECOMMENDED... makes things messy. It is unique from factory
/*
func (p *Sds011) ChangeID(newId byte) error {
	reply, replyErr := p.queryAndWaitResponse(NewPacket_SetId(p.Id, newId))
	if replyErr != nil {
		return replyErr
	}
	target,
}
*/

func (p *Sds011) ChangeToWork(toWork bool) error {
	reply, replyErr := p.queryAndWaitResponse(NewPacket_SetWorkMode(p.Id, true, toWork))
	if replyErr != nil {
		return fmt.Errorf("Change to work failed with %v\n", replyErr.Error())
	}
	target, errGetWork := reply.GetWorkMode()
	if errGetWork != nil {
		return errGetWork
	}
	if target != toWork {
		return fmt.Errorf("Changing work to %v failed", toWork)
	}
	return nil
}

//Not like working/broken.... it means working not sleeping
func (p *Sds011) IsWorking() (bool, error) {
	reply, replyErr := p.queryAndWaitResponse(NewPacket_SetWorkMode(p.Id, false, false))
	if replyErr != nil {
		return false, replyErr
	}
	return reply.GetWorkMode()
}

/*
Called after disconnect
Also allows to query current (desired) settings... even sensor is offline.
Or if it is online.. it returns what setting really is
*/
func (p *Sds011) SyncSettings() (Sds011Settings, error) {
	if p.PassiveMode {
		return p.settings, nil //No activity
	}
	if !p.settingsChanged { //DO not change just read what is going on sensor
		var err error
		p.settings, err = p.readSettings()
		return p.settings, err
	}

	//Reads back.. if are same then no writing
	onSensorSettings, errRead := p.readSettings()
	if errRead != nil {
		return p.settings, errRead
	}
	p.settings.Version = onSensorSettings.Version //Not really setting
	if p.settings.Period != onSensorSettings.Period {
		errWrite := p.writePeriod(p.settings.Period)
		if errWrite != nil {
			return p.settings, errWrite
		}
	}
	if p.settings.QueryMode != onSensorSettings.QueryMode {
		errWrite := p.writeQueryMode(p.settings.QueryMode)
		if errWrite != nil {
			return p.settings, errWrite
		}
	}
	return p.settings, nil
}

/*
Check situation first from sensor. Do not update if not needed, avoid re-flashing eeprom
*/
func (p *Sds011) SetSettings(newSettings Sds011Settings) error {
	if 30 < newSettings.Period {
		return fmt.Errorf("Invalid period %v", newSettings.Period)
	}

	if p.PassiveMode {
		return fmt.Errorf("Write not allowed in passive mode")
	}
	p.settingsChanged = true
	p.settings = newSettings
	return nil
}

//Does reporting non-blocking way. If end user is not intrested errors :(
func (p *Sds011) reportError(err error) {
	if len(p.ErrorsCh) <= cap(p.ErrorsCh) {
		p.ErrorsCh <- err
	}
}

/*

 */
func (p *Sds011) Run() {
	go func() { //two state machine
		for {
			for !p.SettingsInSync {
				time.Sleep(SYNCRETRYINTEVAL * time.Millisecond)
				_, errSync := p.SyncSettings()
				//Report
				p.reportError(errSync)
			}
			for p.SettingsInSync {
				time.Sleep(time.Second)
			}
		}
	}()

	//Handle input
	for {
		pack, ok := <-p.fromSensor
		if !ok {
			return //Channel closed
		}
		if pack.MatchToId(p.Id) && pack.Valid {
			if !p.powerEnable { //Power should be off. Failed power switch or bug in the software
				p.ErrorsCh <- fmt.Errorf("Sensor switch fail, recieved packet %v", pack.ToString())
			}
			switch pack.CommandID {
			case COMMANDID_DATAREPLY:
				measResult, errMeas := pack.GetMeasurement()
				if errMeas == nil {
					//If enough since previous time. Then it is more than extra poll query
					if p.settings.PeriodDuration().Seconds()-1 <= time.Since(p.tPrevResultTime).Seconds() {
						if p.settings.QueryMode {
							//On query mode. One must estimate how many periods have happend
							//Even with the zero communication system can run
							p.measurementCounter += int(math.Floor(time.Since(p.tPrevResultTime).Seconds() / p.settings.PeriodDuration().Seconds()))
						} else {
							p.measurementCounter++ //This is clearly the event. Spontanious sending is more accurate
						}
						p.tPrevResultTime = time.Now()
					}

					//Increase counter. Recieving data does not prove anything.

					measResult.MeasurementCounter = p.measurementCounter
					p.resultCh <- measResult
				}
			case COMMANDID_RESPONSE:
				p.filtreplyFromSensor <- pack
			case COMMANDID_CMD:
				p.reportError(fmt.Errorf("!!!! WARNING SDS011 is recieving in wrong way CMD %v possible RX-TX short", pack.ToString()))

			default: //Should not really happen. Packet filtered earlier in stage. Needed if bad message transfer implemention
				p.reportError(fmt.Errorf("SHOULD NOT HAPPEN bad message transfer implementation INVALID PACKET %v, DISCARDING\n", pack.ToString()))
			}
		} else {
			if pack.Valid {
				if len(p.DetectedSensor) < cap(p.DetectedSensor) {
					p.DetectedSensor <- pack.DeviceID
				}
			} else {
				p.reportError(fmt.Errorf("Discarding packet. Should not happen bad implementation"))
			}
		}
	}
}
