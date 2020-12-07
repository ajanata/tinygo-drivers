package main

import (
	"image/color"
	"machine"
	"time"

	"tinygo.org/x/drivers/rgb75"
)

var disp *rgb75.Device

func main() {

	disp = rgb75.New(
		machine.HUB75_OE, machine.HUB75_LAT, machine.HUB75_CLK,
		[6]machine.Pin{
			machine.HUB75_R1, machine.HUB75_G1, machine.HUB75_B1,
			machine.HUB75_R2, machine.HUB75_G2, machine.HUB75_B2,
		},
		[]machine.Pin{
			machine.HUB75_ADDR_A, machine.HUB75_ADDR_B, machine.HUB75_ADDR_C,
			machine.HUB75_ADDR_D, machine.HUB75_ADDR_E,
		})

	if err := disp.Configure(rgb75.Config{
		Width:      64,
		Height:     32,
		ColorDepth: 4,
	}); nil != err {
		for {
			println("error: " + err.Error())
			time.Sleep(time.Second)
		}
	}

	disp.Resume()
	for {
		for y := int16(0); y < 32; y++ {
			for x := int16(0); x < 64; x++ {
				disp.SetPixel(x, y, color.RGBA{0, 0xFF, 0, 0xFF})
				time.Sleep(10 * time.Millisecond)
				disp.SetPixel(x, y, color.RGBA{0xFF, 0x00, 0, 0xFF})
			}
		}
		disp.ClearDisplay()
	}
}
