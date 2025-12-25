package serial

import (
	"context"
	"fmt"
	"log"

	"math"
	"strings"
	"time"

	"go.bug.st/serial"
)

// PortMock is a special port name that triggers the mock data generator.
const PortMock = "MOCK"

// Device represents your USB EEG hardware.
type Device struct {
	PortName string
	BaudRate int
	// Output: A channel where we will push raw data chunks.
	// Other parts of your app (DSP) will listen to this.
	DataStream chan []byte
}

// New creates a new Device instance.
func New(port string, baud int) *Device {
	return &Device{
		PortName: port,
		BaudRate: baud,
		// Buffer the channel to 100 slots.
		// If the DSP falls slightly behind, the Serial reader won't block immediately.
		DataStream: make(chan []byte, 100),
	}
}

// Start opens the connection and starts the background reading loop.
// It accepts a Context so you can stop it cleanly later.
func (d *Device) Start(ctx context.Context) error {
	// SPECIAL CASE: Mock Mode
	if d.PortName == PortMock {
		go d.mockLoop(ctx)
		return nil
	}

	mode := &serial.Mode{
		BaudRate: d.BaudRate,
	}

	// 1. Open the Port
	port, err := serial.Open(d.PortName, mode)
	if err != nil {
		return fmt.Errorf("failed to open port %s: %w", d.PortName, err)
	}

	// 2. Start the Reader Goroutine
	// This runs in the background forever until the app stops.
	go d.readLoop(ctx, port)

	return nil
}

// readLoop handles the continuous reading of bytes and framing packets.
func (d *Device) readLoop(ctx context.Context, port serial.Port) {
	// Ensure we close the port and the channel when this loop exits
	defer port.Close()
	defer close(d.DataStream)

	log.Printf("🔌 Serial Connected: %s @ %d baud", d.PortName, d.BaudRate)

	// Buffer for raw reads
	readBuf := make([]byte, 1024)
	// Accumulator for building a full packet
	var packetBuf []byte

	for {
		// Check cancellation
		select {
		case <-ctx.Done():
			log.Println("🔌 Stopping Serial Reader...")
			return
		default:
		}

		// Read raw bytes
		n, err := port.Read(readBuf)
		if err != nil {
			log.Printf("❌ Serial Read Error: %v", err)
			return
		}

		if n > 0 {
			// Append new bytes to accumulator
			packetBuf = append(packetBuf, readBuf[:n]...)

			// Process accumulator for valid packets
			// Protocol: 25 bytes total. Byte 0 is 'A' (0x41).
			for len(packetBuf) >= 25 {
				// Find start byte 'A'
				if packetBuf[0] != 'A' {
					// Discard byte (slide window) until we find 'A' or run out
					packetBuf = packetBuf[1:]
					continue
				}

				// We have 'A' at index 0. Check if we have enough bytes for a full frame.
				if len(packetBuf) < 25 {
					// Not enough yet, wait for more data
					break
				}

				// Full packet found! Extract 25 bytes.
				fullPacket := make([]byte, 25)
				copy(fullPacket, packetBuf[:25])

				// Advance accumulator
				packetBuf = packetBuf[25:]

				// Send to DSP
				select {
				case d.DataStream <- fullPacket:
				default:
					log.Println("⚠️ Warning: DSP buffer full, dropping packet")
				}
			}
		}
	}
}

// mockLoop generates synthetic data to simulate a device.
func (d *Device) mockLoop(ctx context.Context) {
	defer close(d.DataStream)
	log.Printf("🔮 Mock Mode Started: Generating synthetic signals...")

	ticker := time.NewTicker(4 * time.Millisecond) // ~250Hz
	defer ticker.Stop()

	t := 0.0
	for {
		select {
		case <-ctx.Done():
			log.Println("🔮 Stopping Mock Reader...")
			return
		case <-ticker.C:
			// Frame: [ 'A' ] [ 3-byte Ch1 ] [ 3-byte Ch2 ] ... [ 3-byte Ch8 ]
			// Total 1 + 24 = 25 bytes.
			// Format: Big Endian Signed 24-bit.
			packet := make([]byte, 25)
			packet[0] = 'A'

			// Generate 8 channels of data
			// We'll vary phases/frequencies slightly so channels look different
			for ch := 0; ch < 8; ch++ {
				// Sine wave: amplitude ~8 million (full 24-bit range is +/- 8.3M)
				// Offset phases by channel index to look cool
				valFloat := 8000000 * math.Sin(t+float64(ch))
				valInt := int32(valFloat)

				// Encode 24-bit Big Endian
				// B0 is MSB, B2 is LSB
				startIndex := 1 + (ch * 3)
				packet[startIndex] = byte((valInt >> 16) & 0xFF)
				packet[startIndex+1] = byte((valInt >> 8) & 0xFF)
				packet[startIndex+2] = byte(valInt & 0xFF)
			}
			t += 0.1

			select {
			case d.DataStream <- packet:
			default:
			}
		}
	}
}

// ListPorts returns a list of available serial ports.
func ListPorts() ([]string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("failed to list serial ports: %w", err)
	}
	return ports, nil
}

// FindPreferredPort searches for a port matching specific patterns.
func FindPreferredPort(ports []string) string {
	for _, p := range ports {
		if strings.Contains(p, "usbmodem") || strings.Contains(p, "usbserial") {
			return p
		}
	}
	return ""
}
