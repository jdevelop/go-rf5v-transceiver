package main

import (
	"github.com/davecheney/gpio"
	"github.com/jdevelop/go-manchester/manchester"
	"time"
	"fmt"
	"flag"
	"sync"
)

func receive(bps int64, pinNum int) {
	pin, err := gpio.OpenPin(pinNum, gpio.ModeInput)
	if err != nil {
		panic(err)
	}

	frame := manchester.NewDataFrame()
	driver := manchester.NewManchesterDriver(bps)

	type Event struct {
		pinState  bool
		eventTime int64
	}

	events := make(chan Event, 1000)

	var latch sync.WaitGroup
	latch.Add(1)

	go func() {
		worker := func(evt Event) {
			var state manchester.Edge
			if evt.pinState {
				state = manchester.Up
			} else {
				state = manchester.Down
			}
			driver.ReadBit(state, evt.eventTime, func(x bool) {
				if frame.ReadBit(x) {
					if frame.IsValid() {
						fmt.Println("============================")
						fmt.Println(string(frame.Data))
						latch.Done()
					} else {
						fmt.Println("Missed")
						frame.Reset()
					}
				}
			})

		}

		for {
			worker(<-events)
		}
	}()

	if err = pin.BeginWatch(gpio.EdgeBoth, func() {
		received := time.Now().UnixNano()
		events <- Event{pin.Get(), received}
	}); err != nil {
		panic(err)
	}

	defer pin.EndWatch()
	defer pin.Close()

	latch.Wait()

}

func main() {
	pin := flag.Int("rp", 2, "Receiver pin")
	bps := flag.Int64("bps", 3, "BPS")
	flag.Parse()
	fmt.Printf("Using PIN %1d and bps speed %2d\n", *pin, *bps)
	receive(*bps, *pin)
}