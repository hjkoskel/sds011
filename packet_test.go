package sds011

import (
	"bytes"
	"testing"
)

func TestPacketExtract(t *testing.T) {
	pack := NewPacket_QueryVersionReply(0xFFFF, 19, 9, 28)
	s, errVer := pack.GetVersionString()
	if errVer != nil {
		t.Errorf("ERROR VERSION %v\n", errVer.Error())
	}
	if s != "19.9.28" {
		t.Errorf("Invalid version string %v", s)
	}
	pack = NewPacket_SetPeriodReply(0xFFFF, true, 7)
	per, _ := pack.GetPeriod()
	if per != 7 {
		t.Errorf("Invalid period %v", per)
	}
	pack = NewPacket_SetWorkModeReply(123, true, true)
	mustWork, _ := pack.GetWorkMode()
	if !mustWork {
		t.Errorf("Work mode parse err")
	}

	pack = NewPacket_SetWorkModeReply(123, true, false)
	mustNotWork, _ := pack.GetWorkMode()
	if mustNotWork {
		t.Errorf("Work mode parse err")
	}

	pack = NewPacket_DataReply(123, 0xABCD, 0x1234)
	measRes, measResErr := pack.GetMeasurement()
	if (measRes.SmallReg != 0xABCD) || (measRes.LargeReg != 0x1234) {
		t.Errorf("invalid meas res resp %#v packet:%#v err:%s", measRes, pack, measResErr)
	}

	pack = NewPacket_SetQueryModeReply(333, true, true)
	querying, _ := pack.GetQueryMode()
	if !querying {
		t.Errorf("Get query not querying")
	}

	pack = NewPacket_SetQueryModeReply(333, true, false)
	notQuerying, _ := pack.GetQueryMode()
	if notQuerying {
		t.Errorf("Get query is querying")
	}

}

func ByteArrayIsEqual(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// Packet conversions from document. "nails down" to something real
// Datasheet examples are valuable as gold. Even they feel stupid
func TestPacketConversionsFromDoc(t *testing.T) {
	/*
		PC sends command, query the current working mode:
		AA B4 02 00 00 00 00 00 00 00 00 00 00 00 00 FF FF 00 AB
	*/
	pack := NewPacket_SetQueryMode(0xFFFF, false, false)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xB4, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0xAB}) {
		t.Errorf("Invalid packet")
	}

	/*
		   Sensor with ID A160 response:
		   AA C5 02 00 00 00 A1 60 03 AB
			 Show the sensor is in report active mode.
	*/
	pack = NewPacket_SetQueryModeReply(0xA160, false, false)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xC5, 0x02, 0x00, 0x00, 0x00, 0xA1, 0x60, 0x03, 0xAB}) {
		t.Errorf("Invalid packet")
	}
	/*
	   AA C5 02 00 01 00 A1 60 04 AB
	   Show the sensor is in report query mode.
	*/
	pack = NewPacket_SetQueryModeReply(0xA160, false, true)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xC5, 0x02, 0x00, 0x01, 0x00, 0xA1, 0x60, 0x04, 0xAB}) {
		t.Errorf("Invalid packet")
	}

	/*
	   PC sends command, set the sensor with ID A160 to report query mode:
	   AA B4 02 01 01 00 00 00 00 00 00 00 00 00 00 A1 60 05 AB
	*/
	pack = NewPacket_SetQueryMode(0xA160, true, true)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xB4, 0x02, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA1, 0x60, 0x05, 0xAB}) {
		t.Errorf("Invalid packet")
	}
	/*
	   Sensor with ID A160 response:
	   AA C5 02 01 01 00 A1 60 05 AB
	   Show the sensor is set to report query mode.
	*/
	pack = NewPacket_SetQueryModeReply(0xA160, true, true)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xC5, 0x02, 0x01, 0x01, 0x00, 0xA1, 0x60, 0x05, 0xAB}) {
		t.Errorf("Invalid packet")
	}

	/*
		Send command:
		AA B4 04 00 00 00 00 00 00 00 00 00 00 00 00 FF FF 02 AB
	*/
	pack = NewPacket_QueryData(0xFFFF)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xB4, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x02, 0xAB}) {
		t.Errorf("Invalid packet")
	}
	/*
	   Sensor with ID A160 Reply:
	   AA C0 D4 04 3A 0A A1 60 1D AB
	   Show PM2.5 data is 04D4, convert it to a decimal 1236,then it show PM2.5 to 123.6μg/m 3 ,PM10
	   data is 0A3A, convert it to a decimal 2618, then it show PM10 to 261.8μg/m 3 .
	*/
	pack = NewPacket_DataReply(0xA160, 1236, 2618)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xC0, 0xD4, 0x04, 0x3A, 0x0A, 0xA1, 0x60, 0x1D, 0xAB}) {
		t.Errorf("Invalid packet")
	}

	/*
		Send command:
		AA B4 04 00 00 00 00 00 00 00 00 00 00 00 00 A1 60 05 AB
	*/
	pack = NewPacket_QueryData(0xA160)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xB4, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA1, 0x60, 0x05, 0xAB}) {
		t.Errorf("Invalid packet")
	}

	/*
		Send command, set the device ID from A160 to A001:
		AA B4 05 00 00 00 00 00 00 00 00 00 00 A0 01 A1 60 A7 AB
	*/
	pack = NewPacket_SetId(0xA160, 0xA001)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xB4, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA0, 0x01, 0xA1, 0x60, 0xA7, 0xAB}) {
		t.Errorf("Invalid packet")
	}
	/*
		Sensor with ID A160 response:
		AA C5 05 00 00 00 A0 01 A6 AB  (mis spaced in datasheet)
	*/
	/*
		pack = NewPacket_SetIdReply(0xA160)
		if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xC5, 0x05, 0x00, 0x00, 0x00, 0xA0, 0x01, 0xA6, 0xAB}) {
			t.Errorf("Invalid packet")
		}
	*/

	/*
		Send command, set the sensor with ID A160 to sleep:
		AA B4 06 01 00 00 00 00 00 00 00 00 00 00 00 A1 60 08 AB
	*/
	pack = NewPacket_SetWorkMode(0xA160, true, false)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xB4, 0x06, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA1, 0x60, 0x08, 0xAB}) {
		t.Errorf("Invalid packet")
	}
	/*
		Sensor with ID A160 response:
		AA C5 06 01 00 00 A1 60 08 AB
	*/
	pack = NewPacket_SetWorkModeReply(0xA160, true, false)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xC5, 0x06, 0x01, 0x00, 0x00, 0xA1, 0x60, 0x08, 0xAB}) {
		t.Errorf("Invalid packet")
	}
	/*
		(2) Send command, set the sensor with ID A160 to work:
		AA B4 06 01 01 00 00 00 00 00 00 00 00 00 00 A1 60 09 AB
	*/
	pack = NewPacket_SetWorkMode(0xA160, true, true)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xB4, 0x06, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA1, 0x60, 0x09, 0xAB}) {
		t.Errorf("Invalid packet")
	}

	/*
		Sensor with ID A160 response:
		AA C5 06 01 01 00 A1 60 09 AB
	*/
	pack = NewPacket_SetWorkModeReply(0xA160, true, true)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xC5, 0x06, 0x01, 0x01, 0x00, 0xA1, 0x60, 0x09, 0xAB}) {
		t.Errorf("Invalid packet")
	}

	/*
		(3) Send command, query the working mode of the sensor with ID A160:
		AA B4 06 00 00 00 00 00 00 00 00 00 00 00 00 A1 60 07 AB
	*/
	pack = NewPacket_SetWorkMode(0xA160, false, false)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xB4, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA1, 0x60, 0x07, 0xAB}) {
		t.Errorf("Invalid packet")
	}
	/*
		Sensor with ID A160 response, show it is in working mode:
		AA C5 06 00 01 00 A1 60 08 AB
	*/
	pack = NewPacket_SetWorkModeReply(0xA160, false, true)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xC5, 0x06, 0x00, 0x01, 0x00, 0xA1, 0x60, 0x08, 0xAB}) {
		t.Errorf("Invalid packet")
	}
	/*
		Or reply:
		AA C5 06 00 00 00 A1 60 07 AB
		show it is in working mode: (LOL.. typo on datasheet)  really bad checksum function btw
	*/
	pack = NewPacket_SetWorkModeReply(0xA160, false, false)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xC5, 0x06, 0x00, 0x00, 0x00, 0xA1, 0x60, 0x07, 0xAB}) {
		t.Errorf("Invalid packet")
	}

	/*
			(1) Send command to set the working period of sensor with ID A160 to 1 minute:
		AA B4 08 01 01 00 00 00 00 00 00 00 00 00 00 A1 60 0B AB
	*/
	pack = NewPacket_SetPeriod(0xA160, true, 1)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xB4, 0x08, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA1, 0x60, 0x0B, 0xAB}) {
		t.Errorf("Invalid packet")
	}
	/*
		Sensor with ID A160 response:
		AA C5 08 01 01 00 A1 60 0B AB Show the sensor will update data in 1 minute.
	*/
	pack = NewPacket_SetPeriodReply(0xA160, true, 1)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xC5, 0x08, 0x01, 0x01, 0x00, 0xA1, 0x60, 0x0B, 0xAB}) {
		t.Errorf("Invalid packet")
	}

	/*
		(2) Send command to set the working period of sensor with ID A160 to 0,it will work
		continuously:
		AA B4 08 01 00 00 00 00 00 00 00 00 00 00 00 A1 60 0A AB
	*/
	pack = NewPacket_SetPeriod(0xA160, true, 0)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xB4, 0x08, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA1, 0x60, 0x0A, 0xAB}) {
		t.Errorf("Invalid packet")
	}

	/*
		Sensor with ID A160 response:
		AA C5 08 01 00 00 A1 60 0A AB
	*/
	pack = NewPacket_SetPeriodReply(0xA160, true, 0)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xC5, 0x08, 0x01, 0x00, 0x00, 0xA1, 0x60, 0x0A, 0xAB}) {
		t.Errorf("Invalid packet")
	}
	/*
		(3) Send command to query the working period of the sensor with ID A160:
		AA B4 08 00 00 00 00 00 00 00 00 00 00 00 00 A1 60 09 AB
	*/
	pack = NewPacket_SetPeriod(0xA160, false, 0)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xB4, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA1, 0x60, 0x09, 0xAB}) {
		t.Errorf("Invalid packet")
	}

	/*
		Sensor with ID A160 response:
		AA C5 08 00 02 00 A1 60 0B AB Show its working period is 2 minute; it will update data every 2 minute.
	*/
	pack = NewPacket_SetPeriodReply(0xA160, false, 2)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xC5, 0x08, 0x00, 0x02, 0x00, 0xA1, 0x60, 0x0B, 0xAB}) {
		t.Errorf("Invalid packet")
	}
	/*
		Send command to get the firmware version of the sensor with ID A160:
		AA B4 07 00 00 00 00 00 00 00 00 00 00 00 00 A1 60 08 AB
	*/
	pack = NewPacket_QueryVersion(0xA160)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xB4, 0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA1, 0x60, 0x08, 0xAB}) {
		t.Errorf("Invalid packet")
	}
	/*
		Sensor with ID A160 response:
		AA C5 07 0F 07 0A A1 60 28 AB
		Show its firmware version is 0F070A(15-7-10).
	*/
	pack = NewPacket_QueryVersionReply(0xA160, 15, 7, 10)
	if !ByteArrayIsEqual(pack.ToBytes(), []byte{0xAA, 0xC5, 0x07, 0x0F, 0x07, 0x0A, 0xA1, 0x60, 0x28, 0xAB}) {
		t.Errorf("Invalid packet")
	}
}

func TestPacketConversions(t *testing.T) {
	testPackets := []Packet{
		NewPacket_SetQueryMode(0xFFFF, true, true),
		NewPacket_SetQueryModeReply(0xFFFF, true, false),
		NewPacket_QueryData(0xFFFF),
		NewPacket_DataReply(0xA160, 0x04D4, 0x0A3A),
		NewPacket_SetId(0xFFFF, 0x1234),
		NewPacket_SetIdReply(0xFFFF),
		NewPacket_SetWorkMode(0xA160, true, false),
		NewPacket_SetWorkModeReply(0xFFFF, true, true),
		NewPacket_SetPeriod(0xA160, true, 1),
		NewPacket_SetPeriodReply(0xA160, false, 2),
		NewPacket_QueryVersion(0xA160),
		NewPacket_QueryVersionReply(0xA160, 0x0F, 0x07, 0x0A),
		NewPacket_QueryVersion(0xABAB),
	}

	for _, pack := range testPackets {
		if !pack.Valid {
			t.Errorf("Packet is not valid %#v\n", pack)
			t.FailNow()
		}

		byteArr := pack.ToBytes()
		switch pack.CommandID {
		case COMMANDID_CMD:
			if len(byteArr) != SDS011TOSENSORSIZE {
				t.Errorf("Invalid packet size TO sensor must be %v not %v", SDS011TOSENSORSIZE, len(byteArr))
			}
		case COMMANDID_RESPONSE, COMMANDID_DATAREPLY:
			if len(byteArr) != SDS011FROMSENSORSIZE {
				t.Errorf("Invalid packet size FROM sensor must be %v not %v", SDS011FROMSENSORSIZE, len(byteArr))
			}

		}
		/*SDS011TOSENSORSIZE   = 19
		SDS011FROMSENSORSIZE = 10*/

		var pack2 Packet
		parseErr := pack2.FromBytes(0, byteArr)
		if parseErr != nil {
			t.Errorf("PARSING ERROR %v\n", parseErr.Error())
			t.FailNow()
		}
		if !pack2.Valid {
			t.Errorf("Packet is not valid %#v\n", pack2)
			t.FailNow()
		}

		byteArr2 := pack2.ToBytes()
		if !bytes.Equal(byteArr2, byteArr) {
			t.Errorf("ERROR NOT EQUAL BYTEARR\n")
		}
	}
}
