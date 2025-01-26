package sds011

import "fmt"

type Result struct {
	MeasurementCounter int
	Uptime             int64
	SmallReg           uint16
	LargeReg           uint16
}

func (p *Result) Small() float64 {
	return float64(p.SmallReg) / 10
}
func (p *Result) Large() float64 {
	return float64(p.LargeReg) / 10
}

// NOTICE: non calibrated values, used for debug
func (p *Result) ToString() string {
	return fmt.Sprintf("count=%v %v PM2.5= %.1fµm/m³ PM10= %.1fµm/m³", p.MeasurementCounter, millisecToString(p.Uptime), p.Small(), p.Large())
}
