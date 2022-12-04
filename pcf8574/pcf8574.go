// Package pcf8574 is a driver for the PCF8574 I2C GPIO expander.
//
// This expander is somewhat limited: Each pin can be set to either high (with a weak pullup) or low (grounded), as well
// as read. To use a pin for input, set it to "high" and check to see if something is forcing it to be low. To use a pin
// for output, they can sink a small amount of current when set to "low".
//
// The chip will generate an interrupt on the rising and falling edge of any input ("high") pin. This is simple enough
// that implementing it (if desired) is left as an exercise to the user.
//
// The PCF8575 is similar but operates on 16 bits instead of 8 bits.
//
// Datasheet: https://cdn-learn.adafruit.com/assets/assets/000/113/910/original/pcf8574.pdf
package pcf8574

import (
	"tinygo.org/x/drivers"
)

const DefaultAddress = 0x20

type Device struct {
	bus  drivers.I2C
	addr uint16
	// current state of pins as we've defined them
	state uint8
}

type Config struct {
	Address uint8
}

type Report uint8

// New creates a new driver on the specified preconfigured I2C bus. The datasheet claims a maximum speed of 100 kHz.
func New(bus drivers.I2C) *Device {
	return &Device{
		bus: bus,
		// defaults to everything high
		state: 0xFF,
	}
}

func (d *Device) Configure(c Config) {
	if c.Address == 0 {
		c.Address = DefaultAddress
	}

	d.addr = uint16(c.Address)
}

// SetPin configures a single pin based on val: True to activate the weak pullup resistor, false to sink current.
func (d *Device) SetPin(pin uint8, val bool) error {
	if val {
		d.state = d.state | 1<<pin
	} else {
		d.state = d.state & ^(1 << pin)
	}
	return d.send()
}

// SetAll configures all pins at once based on their bit in state: True to activate the weak pullup resistor, false to sink current.
func (d *Device) SetAll(state uint8) error {
	d.state = state
	return d.send()
}

func (d *Device) send() error {
	buf := [1]byte{d.state}
	return d.bus.Tx(d.addr, buf[:], nil)
}

// Read reads the status of every pin and returns a Report which can be used to check specific pins.
func (d *Device) Read() (Report, error) {
	var buf [1]byte
	// the chip doesn't have any registers and just returns the data directly when read
	err := d.bus.Tx(d.addr, nil, buf[:])
	return Report(buf[0]), err
}

// Pin reports whether the specified pin is high.
func (r Report) Pin(p uint8) bool {
	return r&(1<<p) > 0
}
