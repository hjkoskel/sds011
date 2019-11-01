/*
Empirical and stolen equations for humidity compensation.
Pick your poison, or edit your own. SDS011 is only interface library. This is just extra collection from internet

I do not have laboratory equipment so can not prove that these work
*/

package sds011

import "math"

/*
Stolen from
https://github.com/piotrkpaul/esp8266-sds011
*/
func NormalizePM25(pm25 float64, humidity float64) float64 {
	return pm25 / (1.0 + 0.48756*math.Pow((humidity/100.0), 8.60068))
}

func NormalizePM10(pm10 float64, humidity float64) float64 {
	return pm10 / (1.0 + 0.81559*math.Pow((humidity/100.0), 5.83411))
}
