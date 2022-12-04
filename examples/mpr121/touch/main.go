package main

import (
	"machine"
	"time"

	"tinygo.org/x/drivers/mpr121"
)

func main() {
	time.Sleep(time.Second)

	err := machine.I2C0.Configure(machine.I2CConfig{
		SDA:       machine.I2C0_SDA_PIN,
		SCL:       machine.I2C0_SCL_PIN,
		Frequency: 400 * machine.KHz,
	})
	if err != nil {
		panic(err)
	}

	dev := mpr121.New(machine.I2C0)
	err = dev.Configure(mpr121.Config{
		Address:          mpr121.DefaultAddress,
		TouchThreshold:   0x10,
		ReleaseThreshold: 0x05,
		// ProximityMode:    mpr121.ProximityModeFour,
		AutoConfig: true,
	})
	if err != nil {
		print("config: ")
		panic(err)
	}

	println("starting")
	for {
		s, err := dev.Status()
		if err != nil {
			panic(err)
		}
		for i := uint8(0); i < 13; i++ {
			t := s.Touched(i)
			if t {
				print("-")
			} else {
				print("_")
			}
		}
		println()
		time.Sleep(time.Millisecond)
	}
}
