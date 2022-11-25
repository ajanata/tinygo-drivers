package pcf8523

const (
	Address           = 0x68 // I2C address for PCF8523
	ClkOutControl     = 0x0F // Timer and CLKOUT control register
	Control1          = 0x00 // Control and status register 1
	Control2          = 0x01 // Control and status register 2
	Control3          = 0x02 // Control and status register 3
	Time              = 0x03 // Time registers starting with seconds
	TimerBFreqControl = 0x12 // Timer B source clock frequency control
	TimerBValue       = 0x13 // Timer B value (number clock periods)
	Offset            = 0x0E // Offset register
	Status            = 0x03 // Status register, also hold seconds
)
