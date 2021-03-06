package main

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/facchinm/go-serial"
	"github.com/judwhite/go-svc/svc"
	"github.com/shirou/gopsutil/cpu"
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

func send(port *serial.SerialPort, pin, state string) error {
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

		for {
			quit := func() bool { // this makes defer work correctly
				defer time.Sleep(2000 * time.Millisecond)

				ports, err := serial.GetPortsList()
				if err != nil {
					log.Println(err)
					return false
				}

				log.Println("Available ports:", ports)
				if len(ports) <= 1 {
					log.Println("Not enough serial ports")
					return false
				}

				mode := &serial.Mode{
					BaudRate: 9600,
				}

				port, err := serial.OpenPort(ports[len(ports)-1], mode)
				if err != nil {
					log.Println(err)
					return false
				}

				defer port.Close()

				time.Sleep(5000 * time.Millisecond)
				send(port, "1", "255")

				cpu.Percent(0, false) // just so it works

			forever:
				for {
					select {
					case message := <-p.quit:
						_ = message // not the prettiest thing in the world
						log.Println("Quit signal received...")
						p.wg.Done()
						return true
					default:
						time.Sleep(500 * time.Millisecond)
						cpuUsage, err := cpu.Percent(0, false)
						if err != nil {
							log.Fatal("Something incredible happened:", err)
						}

						if !p.isService {
							log.Println("Percent:", cpuUsage[0])
						}

						cpuPercent := max(min(int64(cpuUsage[0]*2.55), 255), 0)

						err = send(port, "0", strconv.FormatInt(cpuPercent, 10))
						if err != nil {
							log.Println("Send failed:", err)
							break forever
						}
					}
				}
				return false
			}()
			if quit {
				return
			}
		}
	}()

	return nil
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
