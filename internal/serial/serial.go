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

// readLoop handles the continuous reading of bytes.
func (d *Device) readLoop(ctx context.Context, port serial.Port) {
	// Ensure we close the port and the channel when this loop exits
	defer port.Close()
	defer close(d.DataStream)

	log.Printf("🔌 Serial Connected: %s @ %d baud", d.PortName, d.BaudRate)

	// Create a temporary buffer to hold incoming data
	readBuf := make([]byte, 4096)

	for {
		// Check if we have been told to stop (e.g. CTRL+C)
		select {
		case <-ctx.Done():
			log.Println("🔌 Stopping Serial Reader...")
			return
		default:
			// No stop signal, keep reading
		}

		// 3. Read from Hardware
		// This blocks until data arrives or an error occurs
		n, err := port.Read(readBuf)
		if err != nil {
			log.Printf("❌ Serial Read Error: %v", err)
			return // Exit loop on fatal error (connection lost)
		}

		if n > 0 {
			// 4. Safe Data Copy (CRITICAL STEP)
			// We cannot pass 'readBuf' directly because it will be overwritten
			// by the next loop iteration before the DSP engine finishes using it.
			// We must make a copy of exactly the bytes we received.
			packet := make([]byte, n)
			copy(packet, readBuf[:n])

			// 5. Send to DSP via Channel
			select {
			case d.DataStream <- packet:
				// Successfully sent
			default:
				// Channel is full! DSP is too slow or crashed.
				// We drop the packet to prevent the Serial reader from freezing.
				log.Println("⚠️ Warning: DSP buffer full, dropping packet")
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
			// Generate a fake packet (just a sine wave for demo)
			// Let's pretend we have 8 channels of 3 bytes (24-bit) + header
			// For simplicity: just generating 32 bytes of "noise" + signal
			packet := make([]byte, 32)
			val := byte(127 + 127*math.Sin(t))
			for i := range packet {
				packet[i] = val
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
