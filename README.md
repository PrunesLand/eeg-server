# EEG Server

High-performance Go server for capturing, decoding, and streaming EEG data from custom hardware.

## Features
- **Auto-Detection**: automatically prioritizes `/dev/tty.usbmodem*` and `/dev/tty.usbserial*` ports.
- **Protocol Decoding**: Real-time parsing of 24-bit Big Endian signed integer packets (8 channels).
- **Live Conversion**: Converts raw ADC values to voltage using `(Raw * VRef) / (2^24 * Gain)`.
- **Tunable Gain**: Adjust the signal gain dynamically via HTTP API.
- **Mock Mode**: Synthetic signal generator for testing without hardware.

## Installation

### Prerequisites
- [Go 1.22+](https://go.dev/dl/) installed.

### Setup
1. Clone the repository:
   ```bash
   git clone https://github.com/PrunesLand/eeg-server.git
   cd eeg-server
   ```
2. Install dependencies:
   ```bash
   go mod tidy
   ```

## Usage

### 1. Hardware Mode
Connect your USB EEG device. The server will automatically detect it.
```bash
go run cmd/server/main.go
```
*If multiple ports are found, it selects the first one matching `usbmodem` or `usbserial`. If none are found, it falls back to Mock Mode.*

### 2. Mock Mode
simulate data (sine waves) for testing UI or pipelines:
```bash
go run cmd/server/main.go mock
```

### 3. Build Binary (Optional)
To create a standalone executable (ignored by git):
```bash
go build -o server cmd/server/main.go
./server
```

## API
The server exposes an HTTP API on port `8080` to tune parameters live.

### Gain Control
**Get Current Gain:**
```bash
curl http://localhost:8080/api/gain
```

**Set Gain (e.g., to 1.0):**
```bash
curl -X POST -d '{"gain": 1.0}' http://localhost:8080/api/gain
```
*The console output will immediately reflect the new voltage scaling.*
