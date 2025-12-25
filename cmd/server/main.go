package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/PrunesLand/eeg-server.git/internal/serial"
)

func main() {
	fmt.Println("🧠 EEG Server Starting...")

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
	if selectedPort == "" {
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

	// Create a hex dumper for visualization
	go func() {
		for packet := range device.DataStream {
			fmt.Printf("RX (%d bytes): % X\n", len(packet), packet)
		}
		fmt.Println("Data stream closed.")
	}()

	// Wait for context done
	<-ctx.Done()
	// Give a moment for cleanup logs
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Server stopped.")
}
