package main

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/judwhite/go-svc"
	"github.com/shirou/gopsutil/v3/cpu"
	"go.bug.st/serial"
)

// program implements svc.Service
type program struct {
	wg        sync.WaitGroup
	quit      chan struct{}
	isService bool
}

func main() {
	prg := &program{}

	// Call svc.Run to start your program/service.
	if err := svc.Run(prg); err != nil {
		log.Fatal(err)
	}
}

func (p *program) Init(env svc.Environment) error {
	isService := env.IsWindowsService()
	log.Printf("is win service? %v\n", isService)
	p.isService = isService
	return nil
}

func send(port serial.Port, pin, state string) error {
	_, err := port.Write([]byte(pin + "\n"))
	if err != nil {
		return err
	}

	time.Sleep(100 * time.Millisecond)

	_, err = port.Write([]byte(state + "\n"))
	if err != nil {
		return err
	}

	time.Sleep(100 * time.Millisecond)

	return nil
}

func min(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

func max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

func (p *program) Start() error {
	// The Start method must not block, or Windows may assume your service failed
	// to start. Launch a Goroutine here to do something interesting/blocking.

	p.quit = make(chan struct{})

	p.wg.Add(1)

	go func() {
		log.Println("Starting...")
		defer p.wg.Done()

		for {
			select {
			case <-p.quit:
				log.Println("Quit signal received...")
				return
			default:
				p.runMonitorLoop()
			}
		}
	}()

	return nil
}

func (p *program) runMonitorLoop() {
	defer time.Sleep(2000 * time.Millisecond) // Wait before retrying if connection fails

	ports, err := serial.GetPortsList()
	if err != nil {
		log.Println("Error getting ports list:", err)
		return
	}

	log.Println("Available ports:", ports)
	if len(ports) == 0 {
		log.Println("No serial ports found")
		return
	}

	// Use the last available port (you might want to change this logic)
	portName := ports[len(ports)-1]

	mode := &serial.Mode{
		BaudRate: 9600,
		DataBits: 8,
		StopBits: serial.OneStopBit,
		Parity:   serial.NoParity,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		log.Printf("Error opening port %s: %v\n", portName, err)
		return
	}
	defer port.Close()

	log.Printf("Connected to port: %s\n", portName)

	time.Sleep(2000 * time.Millisecond) // Wait for Arduino to reset

	// Initialize pin 1 to 255
	if err := send(port, "1", "255"); err != nil {
		log.Println("Error during initialization:", err)
		return
	}

	// Warm up CPU percentage calculation
	_, err = cpu.Percent(0, false)
	if err != nil {
		log.Println("Error initializing CPU stats:", err)
		return
	}

	// Main monitoring loop
	for {
		select {
		case <-p.quit:
			return
		default:
			time.Sleep(500 * time.Millisecond)

			cpuUsage, err := cpu.Percent(500*time.Millisecond, false)
			if err != nil {
				log.Println("Error getting CPU usage:", err)
				continue
			}

			if len(cpuUsage) == 0 {
				log.Println("No CPU usage data received")
				continue
			}

			if !p.isService {
				log.Printf("CPU Usage: %.2f%%\n", cpuUsage[0])
			}

			cpuPercent := max(min(int64(cpuUsage[0]*2.55), 255), 0)

			err = send(port, "0", strconv.FormatInt(cpuPercent, 10))
			if err != nil {
				log.Println("Send failed, reconnecting:", err)
				return // Break out of this connection and retry
			}
		}
	}
}

func (p *program) Stop() error {
	// The Stop method is invoked by stopping the Windows service, or by pressing Ctrl+C on the console.
	// This method may block, but it's a good idea to finish quickly or your process may be killed by
	// Windows during a shutdown/reboot. As a general rule you shouldn't rely on graceful shutdown.

	log.Println("Stopping...")
	close(p.quit)
	p.wg.Wait()
	log.Println("Stopped.")
	return nil
}
