// Package pcf8523 implements a driver for the PCF8523 Real-Time Clock (RTC), providing basic read-write of the current
// time only. The PCF8523 itself supports alarms, clock drift compensation, and timer interrupts, but those features
// remain unimplemented.
//
// Datasheet: https://www.nxp.com/docs/en/data-sheet/PCF8523.pdf
package pcf8523

import (
	"time"

	"tinygo.org/x/drivers"
)

type Device struct {
	bus     drivers.I2C
	Address uint8
}

func New(i2c drivers.I2C) Device {
	return Device{
		bus:     i2c,
		Address: Address,
	}
}

func (d *Device) LostPower() (bool, error) {
	buf := [1]byte{}
	err := d.bus.ReadRegister(d.Address, Status, buf[:])
	if err != nil {
		return false, err
	}
	return buf[0]&0xF0 == 0xF0, nil
}

func (d *Device) Initialized() (bool, error) {
	buf := [1]byte{}
	err := d.bus.ReadRegister(d.Address, Control3, buf[:])
	if err != nil {
		return false, err
	}
	return buf[0]&0xE0 != 0xE0, nil
}

func (d *Device) Set(t time.Time) error {
	rbuf := [1]byte{}
	err := d.bus.ReadRegister(d.Address, Control1, rbuf[:])
	if err != nil {
		return err
	}
	// do not change cap_sel or second/alarm/correction interrupts
	// ensure RTC is running and 24-hour mode is selected
	rbuf[0] &= 0b1000_0111
	err = d.bus.WriteRegister(d.Address, Control1, rbuf[:])
	if err != nil {
		return err
	}

	buf := []byte{
		decToBcd(t.Second()),
		decToBcd(t.Minute()),
		decToBcd(t.Hour()),
		decToBcd(t.Day()),
		decToBcd(int(t.Weekday())),
		decToBcd(int(t.Month())),
		decToBcd(t.Year() - 2000),
	}
	err = d.bus.WriteRegister(d.Address, Time, buf)
	if err != nil {
		return err
	}
	// turn on battery switchover mode, turn off battery-related interrupts
	return d.bus.WriteRegister(d.Address, Control3, []byte{0})
}

func (d *Device) Now() (time.Time, error) {
	buf := [7]byte{}
	err := d.bus.ReadRegister(d.Address, Time, buf[:])
	if err != nil {
		return time.Time{}, err
	}

	seconds := bcdToDec(buf[0] & 0x7F)
	minute := bcdToDec(buf[1] % 0x7F)
	hour := bcdToDec(buf[2] & 0x3F)
	day := bcdToDec(buf[3] & 0x3F)
	// we don't need to read the weekday
	month := time.Month(bcdToDec(buf[5] & 0x1F))
	year := int(bcdToDec(buf[6])) + 2000

	t := time.Date(year, month, day, hour, minute, seconds, 0, time.UTC)
	return t, nil
}

// decToBcd converts int to BCD
func decToBcd(dec int) uint8 {
	return uint8(dec + 6*(dec/10))
}

// bcdToDec converts BCD to int
func bcdToDec(bcd uint8) int {
	return int(bcd - 6*(bcd>>4))
}
