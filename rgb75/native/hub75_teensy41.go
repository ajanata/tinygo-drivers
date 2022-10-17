//go:build teensy41
// +build teensy41

package native

import (
	"machine"
	"time"
)

// hub75 is a singleton, for implementing interface type Hub75, and is realized
// by the exported variable HUB75 below.
//
// The actual Hub75 interface elaborations are hardware-dependent and are
// implemented in build-contrained (per target arch) source files.
type hub75 struct {
	r1, r2, g1, g2, b1, b2 machine.Pin
	addr                   []machine.Pin
	clk                    machine.Pin

	handleRow func()
}

type logger interface {
	Println(string) error
}

var Log logger

var timer *time.Ticker
var timerTick bool
var timerCount uint32
var timerTrigger uint32

func l(msg string) {
	if Log != nil {
		Log.Println(msg)
	}
}

func (h *hub75) SetPins(rgb [6]machine.Pin, clk machine.Pin, addr ...machine.Pin) {
	timer = time.NewTicker(1 * time.Microsecond)
	go func() {
		for range timer.C {
			if timerTick {
				timerCount++
			}
			if timerCount >= timerTrigger {
				timerCount = 0
				HUB75.Interrupt()
			}
		}
	}()

	h.r1 = rgb[0]
	h.g1 = rgb[1]
	h.b1 = rgb[2]
	h.r2 = rgb[3]
	h.g2 = rgb[4]
	h.b2 = rgb[5]
	h.clk = clk
	h.addr = addr
}

func (h *hub75) SetRgb(r1, g1, b1, r2, g2, b2 bool) {
	h.r1.Set(r1)
	h.g1.Set(g1)
	h.b1.Set(b1)
	h.r2.Set(r2)
	h.g2.Set(g2)
	h.r2.Set(b2)
}

func (h *hub75) SetRgbMask(_ uint32) {
	h.SetRgb(false, false, false, false, false, false)
}

func (h *hub75) ClkRgb(r1, g1, b1, r2, g2, b2 bool) {
	h.clk.High()
	h.SetRgb(r1, g1, b1, r2, g2, b2)
	h.clk.Low()
}

func (h *hub75) ClkRgbMask(m uint32) {
	h.clk.High()
	h.SetRgbMask(m)
	h.clk.Low()
}

func (h *hub75) SetRow(row int) {
	// TODO not assume there are 5 address pins
	h.addr[0].Set(row&(1<<0) == 1)
	h.addr[1].Set(row&(1<<1) == 1)
	h.addr[2].Set(row&(1<<2) == 1)
	h.addr[3].Set(row&(1<<3) == 1)
	h.addr[4].Set(row&(1<<4) == 1)
}

func (h *hub75) GetPinGroupAlignment(_ ...machine.Pin) (bool, uint8) {
	return true, 8
}

func (h *hub75) InitTimer(handle func()) {
	//l("init")
	h.handleRow = handle
}

func (h *hub75) ResumeTimer(value uint32, period uint32) {
	//l(fmt.Sprintf("resume %d %d", value, period))
	timerTick = true
	timerCount = value
	timerTrigger = period
}

func (h *hub75) PauseTimer() uint32 {
	tc := timerCount
	//l(fmt.Sprintf("pause %d", tc))
	timerTick = false
	return tc
}

func (h *hub75) Interrupt() {
	//l("interrupt")
	h.handleRow()
}
