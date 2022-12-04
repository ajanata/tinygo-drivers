package main

import (
	"machine"
	"time"

	"tinygo.org/x/drivers/pcf8574"
)

const intPin = machine.NoPin // change as needed

var dev *pcf8574.Device
var hadIRQ = false
var toggle = false

// bits 0 and 1 are connected to buttons.
// bit 6 is toggled every time bit 0 is low while the data is read (on interrupt or on timer, if there is not interrupt)
// bit 7 mirrors bit 1.

func main() {
	time.Sleep(time.Second)

	err := machine.I2C0.Configure(machine.I2CConfig{
		Frequency: 100 * machine.KHz,
	})
	if err != nil {
		panic(err)
	}

	dev = pcf8574.New(machine.I2C0)
	dev.Configure(pcf8574.Config{})

	if intPin != machine.NoPin {
		intPin.Configure(machine.PinConfig{
			Mode: machine.PinInputPullup,
		})
		err = intPin.SetInterrupt(machine.PinFalling, irq)
		if err != nil {
			panic(err)
		}
		println("interrupt enabled")
	}

	for {
		if hadIRQ || intPin == machine.NoPin {
			hadIRQ = false
			s, err := dev.Read()
			if err != nil {
				panic(err)
			}
			for i := uint8(0); i < 8; i++ {
				if s.Pin(i) {
					print("-")
				} else {
					print("_")
				}
			}
			println()

			if !s.Pin(0) {
				toggle = !toggle
				err = dev.SetPin(6, toggle)
				if err != nil {
					panic(err)
				}
			}

			err = dev.SetPin(7, s.Pin(1))
			if err != nil {
				panic(err)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func irq(_ machine.Pin) {
	println("interrupt")
	hadIRQ = true
}
