package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/PrunesLand/eeg-server.git/internal/api"
	"github.com/PrunesLand/eeg-server.git/internal/serial"
	"github.com/PrunesLand/eeg-server.git/internal/settings"
)

func main() {
	fmt.Println("🧠 EEG Server Starting...")

	// 0. Initialize Settings & API
	appSettings := settings.New()
	go api.StartServer(appSettings)

	// 1. List Ports
	ports, err := serial.ListPorts()
	if err != nil {
		log.Fatalf("Failed to list ports: %v", err)
	}

	if len(ports) == 0 {
		fmt.Println("No serial ports found!")
		fmt.Println("Available ports logic: (mocking if none found for testing could be added here)")
		// For now, valid to just exit or ask user to check connection
		return
	}

	fmt.Println("Available Ports:")
	for i, p := range ports {
		fmt.Printf(" [%d] %s\n", i, p)
	}

	// Simple selection: Pick the first one for now, or let user type?
	// For "one-shot" allow testing, let's pick the first one automatically or use a hardcoded default if preferred.
	// But let's ask user to confirm if we want to be fancy. For now, simple: use the first one.
	selectedPort := serial.FindPreferredPort(ports)

	// Check for manual override
	if len(os.Args) > 1 && os.Args[1] == "mock" {
		fmt.Println("⚠️ Manual override: Switching to MOCK MODE.")
		selectedPort = serial.PortMock
	} else if selectedPort == "" {
		fmt.Println("⚠️ No 'usbmodem' or 'usbserial' found. Switching to MOCK MODE.")
		selectedPort = serial.PortMock
	} else {
		fmt.Printf("✅ Auto-selected Port: %s\n", selectedPort)
	}

	// 2. Start Connection
	// Create a context that we can cancel on interrupt (Ctrl+C)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fmt.Println("\nReceived Ctrl+C, shutting down...")
		cancel()
	}()

	baudRate := 2086956 // baud rate of the eeg device
	device := serial.New(selectedPort, baudRate)

	fmt.Printf("Connecting to %s at %d baud...\n", selectedPort, baudRate)
	if err := device.Start(ctx); err != nil {
		log.Fatalf("Failed to start device: %v", err)
	}

	// 3. Read Data
	fmt.Println("Listening for data... (Press Ctrl+C to stop)")

	// Create a parser
	go func() {
		for packet := range device.DataStream {
			if len(packet) != 25 {
				log.Printf("⚠️ Invalid packet length: %d", len(packet))
				continue
			}

			if packet[0] != 'A' {
				log.Printf("⚠️ Invalid header: %02x", packet[0])
				continue
			}

			// Get current gain dynamically
			currentGain := appSettings.GetGain()
			// Constant Scale Factor = Vref / (2^24)
			// Voltage = (Raw * Scale) / Gain
			const vRef = 5.0
			const maxADC = 16777216.0 // 2^24
			scale := vRef / maxADC

			// Parse 8 channels
			fmt.Print("RX: ")
			for ch := 0; ch < 8; ch++ {
				// 3 bytes per channel (Big Endian)
				start := 1 + (ch * 3)
				b0 := packet[start]
				b1 := packet[start+1]
				b2 := packet[start+2]

				// Reassemble 24-bit Int
				// uint32 first to shift
				val32 := uint32(b0)<<16 | uint32(b1)<<8 | uint32(b2)

				// Sign extension for 24-bit to 32-bit
				if val32&0x800000 != 0 {
					val32 |= 0xFF000000
				}

				// Convert to signed int in Go
				valSigned := int32(val32)

				// Apply Conversion Formula
				// volts = (raw * 5.0) / (2^24 * gain)
				volts := (float64(valSigned) * scale) / currentGain

				fmt.Printf("[%d]: %10.6f V  ", ch+1, volts)
			}
			fmt.Println()
		}
		fmt.Println("Data stream closed.")
	}()

	// Wait for context done
	<-ctx.Done()
	// Give a moment for cleanup logs
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Server stopped.")
}
