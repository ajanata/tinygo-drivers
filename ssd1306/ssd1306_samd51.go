//go:build atsamd51
// +build atsamd51

package ssd1306

import (
	"device/sam"
	"machine"
	"runtime"
	"runtime/interrupt"
	"sync/atomic"
	"unsafe"
)

type DMAConfig struct {
	// DMAChannel is the DMA channel to use.
	DMAChannel uint8
	// DMADescriptor is the descriptor for the specified DMA channel.
	DMADescriptor *DMADescriptor
	// TriggerSource is the DMA trigger source, e.g. SERCOM5_DMAC_ID_TX = 0x0F. This must be for the sercom used for
	// bus in the constructor.
	TriggerSource uint32
}

type I2CDMABus struct {
	dev           *Device
	wire          *machine.I2C
	Address       uint16
	dmaDescriptor *DMADescriptor
	dmaChannel    uint8
	dmaBuf        []byte
	active        atomic.Bool

	cfg *DMAConfig
}

type DMADescriptor struct {
	Btctrl   uint16
	Btcnt    uint16
	Srcaddr  unsafe.Pointer
	Dstaddr  unsafe.Pointer
	Descaddr unsafe.Pointer
}

// NewI2CDMA creates a new driver using I2C with DMA for data transfers (but not command transfers).
// The DMA controller must be properly initialized first.
// The interrupt handler must be hooked up by the caller; see TXComplete. If you fail to do this, the second call to
// Display will hang.
// If the I2C bus is to be used with other peripherals, ensure that Busy returns false before using it.
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
		Dstaddr: unsafe.Pointer(&b.wire.Bus.DATA.Reg),
	}

	// Reset channel.
	sam.DMAC.CHANNEL[b.dmaChannel].CHCTRLA.ClearBits(sam.DMAC_CHANNEL_CHCTRLA_ENABLE)
	sam.DMAC.CHANNEL[b.dmaChannel].CHCTRLA.SetBits(sam.DMAC_CHANNEL_CHCTRLA_SWRST)

	// Configure channel.
	sam.DMAC.CHANNEL[b.dmaChannel].CHPRILVL.Set(0)
	sam.DMAC.CHANNEL[b.dmaChannel].CHCTRLA.Set((sam.DMAC_CHANNEL_CHCTRLA_TRIGACT_BURST << sam.DMAC_CHANNEL_CHCTRLA_TRIGACT_Pos) | (b.cfg.TriggerSource << sam.DMAC_CHANNEL_CHCTRLA_TRIGSRC_Pos) | (sam.DMAC_CHANNEL_CHCTRLA_BURSTLEN_SINGLE << sam.DMAC_CHANNEL_CHCTRLA_BURSTLEN_Pos))

	// Enable transfer complete interrupt.
	sam.DMAC.CHANNEL[b.dmaChannel].CHINTENSET.Set(sam.DMAC_CHANNEL_CHINTENSET_TCMPL)

	// we don't need this anymore, so let it get GC'd
	b.cfg = nil
}

func (b *I2CDMABus) tx(data []byte, isCommand bool) {
	// check for in-flight transfers before commands as well so the DMA doesn't mess with the data being sent by the command
	for b.active.Load() {
		runtime.Gosched()
	}
	if isCommand {
		// use synchronous, slow communication for commands since we have to wait for execution anyway
		b.wire.WriteRegister(uint8(b.Address), 0x00, data)
	} else {
		b.active.Store(true)

		// fire the data via DMA
		b.wire.Bus.ADDR.Set(uint32(b.Address << 1))

		// For some reason, you have to provide the address just past the end of the
		// array instead of the address of the array.
		b.dmaDescriptor.Srcaddr = unsafe.Pointer(uintptr(unsafe.Pointer(&b.dmaBuf[0])) + uintptr(len(b.dmaBuf)))
		b.dmaDescriptor.Btcnt = uint16(len(b.dmaBuf)) // beat count

		// Start the transfer.
		sam.DMAC.CHANNEL[b.dmaChannel].CHCTRLA.SetBits(sam.DMAC_CHANNEL_CHCTRLA_ENABLE)
	}
}

func (b *I2CDMABus) setAddress(address uint16) {
	b.Address = address
}

func (b *I2CDMABus) busy() bool {
	return b.active.Load()
}

// TXComplete is the interrupt handler for DMA transfer completions. You must hook this up yourself with something like
//
//	i2cInt := interrupt.New(sam.IRQ_DMAC_1, dispDMAInt)
//
//	func dispDMAInt(i interrupt.Interrupt) {
//		disp.TXComplete(i)
//	}
//
// from your code (using the appropriate interrupt number). This must be done by you and not here because the interrupt
// number is required to be a constant by the compiler. You must define a function instead of using one inline because
// the compiler does not allow closures for interrupt handlers. Also ensure to enable that interrupt and possibly set
// its priority.
func (d *Device) TXComplete(_ interrupt.Interrupt) {
	b := d.bus.(*I2CDMABus)
	sam.DMAC.CHANNEL[b.dmaChannel].SetCHINTFLAG_TCMPL(1)
	b.active.Store(false)
}
