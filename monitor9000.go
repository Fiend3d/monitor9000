package main

import (
	"log"
	"sync"
	"time"
	"strconv"

	"github.com/judwhite/go-svc/svc"
	"github.com/shirou/gopsutil/cpu"
	"github.com/facchinm/go-serial"
)

// program implements svc.Service
type program struct {
	wg   sync.WaitGroup
	quit chan struct{}
}

func main() {
	prg := &program{}

	// Call svc.Run to start your program/service.
	if err := svc.Run(prg); err != nil {
		log.Fatal(err)
	}
}

func (p *program) Init(env svc.Environment) error {
	log.Printf("is win service? %v\n", env.IsWindowsService())
	return nil
}

func send(port *serial.SerialPort, pin, state string) {
	port.Write([]byte(pin + "\n"))
	time.Sleep(100 * time.Millisecond)
	port.Write([]byte(state + "\n"))
	time.Sleep(100 * time.Millisecond)
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

	p.wg.Add(2)
	go func() {
		log.Println("Starting...")

		go func() {
			ports, err := serial.GetPortsList()
			if err != nil {
				log.Fatal(err)
				p.wg.Done()
				return
			}

			log.Println("Available ports:", ports)
			if len(ports) <= 1 {
				log.Fatal("Not enough serial ports")
				p.wg.Done()
				return
			}

			mode := &serial.Mode{
				BaudRate: 9600,
			}

			port, err := serial.OpenPort(ports[len(ports)-1], mode)
			if err != nil {
				log.Fatal(err)
				p.wg.Done()
				return
			}

			defer port.Close()

			time.Sleep(5000 * time.Millisecond)
			send(port, "1", "255")

			cpu.Percent(0, false) // just so it works 

			for {
				select {
				case message := <-p.quit:
					_ = message // not the prettiest thing in th world
					log.Println("Stopping loop...")
					p.wg.Done()
					return
				default:
					time.Sleep(500 * time.Millisecond)
					cpuUsage, err := cpu.Percent(0, false)
					if err != nil {
						log.Fatal(err)
						p.wg.Done()
						return
					}
					log.Println("Percent:", cpuUsage[0])
					cpuPercent := max(min(int64(cpuUsage[0] * 2.55), 255), 0)
					send(port, "0", strconv.FormatInt(cpuPercent, 10))
					// log.Println("Int:", cpuPercent)
				}
			}
		}()

		<-p.quit
		log.Println("Quit signal received...")
		p.wg.Done()
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
