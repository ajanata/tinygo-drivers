package main

import (
	"machine"
	"time"

	"tinygo.org/x/drivers/pcf8523"
)

var (
	i2c = machine.I2C0
	rtc = pcf8523.New(i2c)
)

// example that will set the clock to a static value if it wasn't already set. if it was set, simply shows the time
func main() {
	time.Sleep(time.Second)
	i2c.Configure(machine.I2CConfig{Frequency: machine.MHz})

	lost, err := rtc.LostPower()
	if err != nil {
		panic(err)
	}
	init, err := rtc.Initialized()
	if err != nil {
		panic(err)
	}
	if lost || !init {
		println("RTC not initialized, setting clock")
		// need to wait 2 seconds after applying power before setting clock
		time.Sleep(2 * time.Second)
		err := rtc.Set(time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC))
		if err != nil {
			panic(err)
		}
	}

	prev := -1

	for {
		for {
			t, err := rtc.Now()
			if err != nil {
				panic(err)
			}
			if prev != t.Second() {
				println(t.String())
				prev = t.Second()
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}
