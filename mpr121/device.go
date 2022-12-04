// Package mpr121 provides a driver for the MPR121 capacitive touch sensor.
//
// Datasheet: https://cdn-shop.adafruit.com/datasheets/MPR121.pdf
package mpr121

import (
	"time"

	"tinygo.org/x/drivers"
)

type Device struct {
	addr uint8
	bus  drivers.I2C
}

type Config struct {
	Address          uint8
	TouchThreshold   uint8
	ReleaseThreshold uint8
	ProximityMode    ProximityMode
	// AutoConfig enables the device's own automatic configuration. Seems to work well enough, you probably want this.
	AutoConfig bool
}

type Report uint16

// ProximityMode indicates how many channels are bundled together for the proximity sensor (starting from the first channel).
type ProximityMode uint8

const (
	ProximityModeOff ProximityMode = iota
	ProximityModeTwo
	ProximityModeFour
	ProximityModeTwelve
)

// New creates a new MPR121 driver on the provided I2C bus. The datasheet says it doesn't support more than 400 kHz, but
// mine worked with 1 MHz. YMMV.
func New(bus drivers.I2C) *Device {
	return &Device{
		bus: bus,
	}
}

func (d *Device) Configure(c Config) error {
	if c.Address == 0 {
		c.Address = DefaultAddress
	}

	d.addr = c.Address
	err := d.write(MPR121_SOFTRESET, 0x63)
	if err != nil {
		return err
	}
	time.Sleep(time.Millisecond)

	err = d.write(MPR121_ECR, 0)
	if err != nil {
		return err
	}

	// should be able to check this, but mine just... doesn't return this properly but otherwise works
	// valid, err := d.read8(MPR121_CONFIG2)
	// if err != nil {
	// 	return err
	// }
	// // default value after reset
	// if valid != 0x24 {
	// 	return errors.New("MPR121 not detected")
	// }

	err = d.SetThresholds(c.TouchThreshold, c.ReleaseThreshold)
	if err != nil {
		return err
	}

	err = d.write(MPR121_MHDR, 0x01)
	if err != nil {
		return err
	}
	err = d.write(MPR121_NHDR, 0x01)
	if err != nil {
		return err
	}
	err = d.write(MPR121_NCLR, 0x0E)
	if err != nil {
		return err
	}
	err = d.write(MPR121_FDLR, 0x00)
	if err != nil {
		return err
	}

	err = d.write(MPR121_MHDF, 0x01)
	if err != nil {
		return err
	}
	err = d.write(MPR121_NHDF, 0x05)
	if err != nil {
		return err
	}
	err = d.write(MPR121_NCLF, 0x01)
	if err != nil {
		return err
	}
	err = d.write(MPR121_FDLF, 0x00)
	if err != nil {
		return err
	}

	err = d.write(MPR121_NHDT, 0x00)
	if err != nil {
		return err
	}
	err = d.write(MPR121_NCLT, 0x00)
	if err != nil {
		return err
	}
	err = d.write(MPR121_FDLT, 0x00)
	if err != nil {
		return err
	}

	err = d.write(MPR121_DEBOUNCE, 0)
	if err != nil {
		return err
	}
	err = d.write(MPR121_CONFIG1, 0x10) // default, 16uA charge current
	if err != nil {
		return err
	}
	err = d.write(MPR121_CONFIG2, 0x20) // 0.5uS encoding, 1ms period
	if err != nil {
		return err
	}

	if c.AutoConfig {
		err = d.write(MPR121_AUTOCONFIG0, 0x0B)
		if err != nil {
			return err
		}

		// correct values for Vdd = 3.3V
		err = d.write(MPR121_UPLIMIT, 200) // ((Vdd - 0.7)/Vdd) * 256
		if err != nil {
			return err
		}
		err = d.write(MPR121_TARGETLIMIT, 180) // UPLIMIT * 0.9
		if err != nil {
			return err
		}
		err = d.write(MPR121_LOWLIMIT, 130) // UPLIMIT * 0.65
		if err != nil {
			return err
		}
	}

	// mask off invalid bits and shift into correct position
	pm := uint8((c.ProximityMode & 0b11) << 4)

	// enable all 12 normal channels plus selected proximity mode
	return d.write(MPR121_ECR, 0b1000_0000+12+pm)
}

// Status reads the state of every touch sensor and returns a Report which can be used to check each channel with a
// single round-trip I2C transaction.
func (d *Device) Status() (Report, error) {
	raw, err := d.read16(MPR121_TOUCHSTATUS_L)
	return Report(raw), err
}

func (r Report) Touched(channel uint8) bool {
	return r&(1<<channel) > 0
}

// SetThresholds sets every channel to the specified thresholds.
//
// Threshold settings are dependent on the touch/release signal strength, system sensitivity and noise immunity requirements. In
// a typical touch detection application, threshold is typically in the range 0x04~0x10. The touch threshold is several counts larger
// than the release threshold. This is to provide hysteresis and to prevent noise and jitter. For more information, refer to the
// application note AN3892 and the MPR121 design guidelines.
func (d *Device) SetThresholds(touch, release uint8) error {
	for i := uint8(0); i <= 12; i++ {
		err := d.SetThreshold(i, touch, release)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetThreshold sets the given channel to the specified thresholds.
//
// Threshold settings are dependent on the touch/release signal strength, system sensitivity and noise immunity requirements. In
// a typical touch detection application, threshold is typically in the range 0x04~0x10. The touch threshold is several counts larger
// than the release threshold. This is to provide hysteresis and to prevent noise and jitter. For more information, refer to the
// application note AN3892 and the MPR121 design guidelines.
func (d *Device) SetThreshold(channel, touch, release uint8) error {
	err := d.write(MPR121_TOUCHTH_0+2*channel, touch)
	if err != nil {
		return err
	}
	return d.write(MPR121_RELEASETH_0+2*channel, release)
}

func (d *Device) read8(reg uint8) (uint8, error) {
	buf := [1]byte{}
	err := d.bus.ReadRegister(d.addr, reg, buf[:])
	return buf[0], err
}

func (d *Device) read16(reg uint8) (uint16, error) {
	buf := [2]byte{}
	err := d.bus.ReadRegister(d.addr, reg, buf[:])
	return (uint16(buf[1]) << 8) | uint16(buf[0]), err
}

func (d *Device) write(reg, val uint8) error {
	// must stop to write most registers
	mustStop := true
	var ecrBackup uint8
	buf := [1]byte{}

	if (reg == MPR121_ECR) || ((0x73 <= reg) && (reg <= 0x7A)) {
		mustStop = false
	}

	if mustStop {
		err := d.bus.ReadRegister(d.addr, MPR121_ECR, buf[:])
		if err != nil {
			return err
		}

		if buf[0] != 0 {
			ecrBackup = buf[0]
			buf[0] = 0
			err = d.bus.WriteRegister(d.addr, MPR121_ECR, buf[:])
			if err != nil {
				return err
			}
		}
	}

	buf[0] = val
	err := d.bus.WriteRegister(d.addr, reg, buf[:])
	if err != nil {
		return err
	}

	if mustStop && ecrBackup != 0 {
		buf[0] = ecrBackup
		err := d.bus.WriteRegister(d.addr, MPR121_ECR, buf[:])
		if err != nil {
			return err
		}
	}

	return nil
}
