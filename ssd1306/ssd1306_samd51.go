//go:build atsamd51
// +build atsamd51

package ssd1306

import (
	"device/sam"
	"machine"
	"unsafe"
)

type DMAConfig struct {
	DMADescriptor *DMADescriptor
	DMAChannel    uint8
	TriggerSource uint32
}

type I2CDMABus struct {
	dev           *Device
	wire          *machine.I2C
	Address       uint16
	dmaDescriptor *DMADescriptor
	dmaChannel    uint8
	dmaBuf        []byte

	cfg *DMAConfig
}

type DMADescriptor struct {
	Btctrl   uint16
	Btcnt    uint16
	Srcaddr  unsafe.Pointer
	Dstaddr  unsafe.Pointer
	Descaddr unsafe.Pointer
}

// NewI2CDMA creates a new driver using I2C over DMA for data transfers (but not command transfers).
// Currently, overlapped transfers are not prevented. Callers should ensure to not call Display() too often or corruption
// of the display will likely occur.
// DMA must be properly initialized first.
func NewI2CDMA(bus *machine.I2C, cfg *DMAConfig) *Device {
	b := &I2CDMABus{
		wire:    bus,
		Address: Address,
		cfg:     cfg,
	}
	dev := &Device{
		bus: b,
	}
	// circular references... annoying but not sure how to work around
	b.dev = dev

	return dev
}

func (b *I2CDMABus) configure() {
	b.dmaDescriptor = b.cfg.DMADescriptor
	b.dmaChannel = b.cfg.DMAChannel
	// buffer needs to be 1 byte larger to have the screen data register address before it
	b.dmaBuf = make([]byte, b.dev.bufferSize+1)
	// we will only be DMAing to the screen data register
	b.dmaBuf[0] = 0x40
	// use a slice of the DMA buffer ignoring the register address byte for the framebuffer to save memory and copy time
	b.dev.buffer = b.dmaBuf[1:]

	*b.dmaDescriptor = DMADescriptor{
		Btctrl: (1 << 0) | // VALID: Descriptor Valid
			(0 << 3) | // BLOCKACT=NOACT: Block Action
			(1 << 10) | // SRCINC: Source Address Increment Enable
			(0 << 11) | // DSTINC: Destination Address Increment Enable
			(1 << 12) | // STEPSEL=SRC: Step Selection
			(0 << 13), // STEPSIZE=X1: Address Increment Step Size
		Dstaddr: unsafe.Pointer(&machine.I2C0.Bus.DATA.Reg),
	}

	// Reset channel.
	sam.DMAC.CHANNEL[b.dmaChannel].CHCTRLA.ClearBits(sam.DMAC_CHANNEL_CHCTRLA_ENABLE)
	sam.DMAC.CHANNEL[b.dmaChannel].CHCTRLA.SetBits(sam.DMAC_CHANNEL_CHCTRLA_SWRST)

	// Configure channel.
	sam.DMAC.CHANNEL[b.dmaChannel].CHPRILVL.Set(0)
	sam.DMAC.CHANNEL[b.dmaChannel].CHCTRLA.Set((sam.DMAC_CHANNEL_CHCTRLA_TRIGACT_BURST << sam.DMAC_CHANNEL_CHCTRLA_TRIGACT_Pos) | (b.cfg.TriggerSource << sam.DMAC_CHANNEL_CHCTRLA_TRIGSRC_Pos) | (sam.DMAC_CHANNEL_CHCTRLA_BURSTLEN_SINGLE << sam.DMAC_CHANNEL_CHCTRLA_BURSTLEN_Pos))

	// we don't need this anymore, so let it get GC'd
	b.cfg = nil
}

func (b *I2CDMABus) tx(data []byte, isCommand bool) {
	if isCommand {
		// use synchronous, slow communication for commands since we have to wait for execution anyway
		b.wire.WriteRegister(uint8(b.Address), 0x00, data)
	} else {
		// fire the data via DMA
		b.wire.Bus.ADDR.Set(uint32(b.Address << 1))

		// For some reason, you have to provide the address just past the end of the
		// array instead of the address of the array.
		b.dmaDescriptor.Srcaddr = unsafe.Pointer(uintptr(unsafe.Pointer(&b.dmaBuf[0])) + uintptr(len(b.dmaBuf)))
		b.dmaDescriptor.Btcnt = uint16(len(b.dmaBuf)) // beat count

		// Start the transfer.
		sam.DMAC.CHANNEL[b.dmaChannel].CHCTRLA.SetBits(sam.DMAC_CHANNEL_CHCTRLA_ENABLE)

		// TODO interrupt when the transfer is over, so we don't overlap transfers
	}
}

func (b *I2CDMABus) setAddress(address uint16) {
	b.Address = address
}
