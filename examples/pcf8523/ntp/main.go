package main

import (
	"errors"
	"fmt"
	"machine"
	"time"

	"tinygo.org/x/drivers/net"
	"tinygo.org/x/drivers/pcf8523"
	"tinygo.org/x/drivers/wifinina"
)

const (
	wifiSSID     = "litterbox"
	wifiPassword = "internet OF shit"
	ntpHost      = "time.nist.gov"
)

var (
	i2c = machine.I2C0
	rtc = pcf8523.New(i2c)
)

// example that will set the time to a value obtained from an NTP server using a wifinina coprocessor.
// if the RTC was already set, does not re-set the time and just shows the time.
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
		println("RTC not initialized, setting clock from ntp")
		ntp()
	} else {
        println("RTC already set")
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

const ntpPacketSize = 48

var b = make([]byte, ntpPacketSize)

// based on https://github.com/tinygo-org/drivers/blob/release/examples/wifinina/ntpclient/main.go
func ntp() {
	err := machine.NINA_SPI.Configure(machine.SPIConfig{
		Frequency: 8 * machine.MHz,
		SDO:       machine.NINA_SDO,
		SDI:       machine.NINA_SDI,
		SCK:       machine.NINA_SCK,
	})
	if err != nil {
		panic(err)
	}

	wifi := wifinina.New(machine.NINA_SPI,
		machine.NINA_CS,
		machine.NINA_ACK,
		machine.NINA_GPIO0,
		machine.NINA_RESETN)
	wifi.Configure()
	time.Sleep(1 * time.Second)

	err = wifi.ConnectToAccessPoint(wifiSSID, wifiPassword, 10*time.Second)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second)
	_, _, _, err = wifi.GetIP()
	if err != nil {
		panic(err)
	}

	// now make UDP connection
	ip := net.ParseIP(ntpHost)
	raddr := &net.UDPAddr{IP: ip, Port: 123}
	laddr := &net.UDPAddr{Port: 2390}
	conn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		panic(err)
	}
	t, err := getCurrentTime(conn)
	if err != nil {
		panic(err)
	}
	err = rtc.Set(t)
	if err != nil {
		panic(err)
	}
}

func getCurrentTime(conn *net.UDPSerialConn) (time.Time, error) {
	if err := sendNTPpacket(conn); err != nil {
		return time.Time{}, err
	}
	clearBuffer()
	for now := time.Now(); time.Since(now) < time.Second; {
		time.Sleep(5 * time.Millisecond)
		if n, err := conn.Read(b); err != nil {
			return time.Time{}, fmt.Errorf("error reading UDP packet: %w", err)
		} else if n == 0 {
			continue // no packet received yet
		} else if n != ntpPacketSize {
			if n != ntpPacketSize {
				return time.Time{}, fmt.Errorf("expected NTP packet size of %d: %d", ntpPacketSize, n)
			}
		}
		return parseNTPpacket(), nil
	}
	return time.Time{}, errors.New("no packet received after 1 second")
}

func clearBuffer() {
	for i := range b {
		b[i] = 0
	}
}

func sendNTPpacket(conn *net.UDPSerialConn) error {
	clearBuffer()
	b[0] = 0b11100011 // LI, Version, Mode
	b[1] = 0          // Stratum, or type of clock
	b[2] = 6          // Polling Interval
	b[3] = 0xEC       // Peer Clock Precision
	// 8 bytes of zero for Root Delay & Root Dispersion
	b[12] = 49
	b[13] = 0x4E
	b[14] = 49
	b[15] = 52
	if _, err := conn.Write(b); err != nil {
		panic(err)
	}
	return nil
}

func parseNTPpacket() time.Time {
	// the timestamp starts at byte 40 of the received packet and is four bytes,
	// this is NTP time (seconds since Jan 1 1900):
	t := uint32(b[40])<<24 | uint32(b[41])<<16 | uint32(b[42])<<8 | uint32(b[43])
	const seventyYears = 2208988800
	return time.Unix(int64(t-seventyYears), 0)
}
