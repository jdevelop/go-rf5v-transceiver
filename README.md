# Go transceiver interface for [RF-5V](https://sites.google.com/site/summerfuelrobots/arduino-sensor-tutorials/rf-wireless-transmitter-receiver-module-433mhz-for-arduino)


The goal of the project is to provide simple [burst packet transmission](https://en.wikipedia.org/wiki/Frame-bursting)

The [Manchester encoding](http://www.atmel.com/Images/Atmel-9164-Manchester-Coding-Basics_Application-Note.pdf) is used to pass data through.

### Usage:

#### Sending data
```golang
    // create the Manchester driver and provide the BPS (bits per second)
	driver := manchester.NewManchesterDriver(*bps)
	// create the data frame
	frame := manchester.BuildDataFrame([]byte(*text))

    // open the pin on GPIO using your favorite library
	pin, _ := gpio.OpenPin(*senderPin, gpio.ModeOutput)

	defer pin.Close()

    // the writer function will be called by the driver to set on/off state 
    // for the pin
	writer := func(level bool) {
		if level {
			pin.Set()
		} else {
			pin.Clear()
		}
	}

    // this is the set of impulses to reduce the noise
	for i := 1; i < 50; i++ {
		pin.Set()
		driver.Sleep()
		pin.Clear()
		driver.Sleep()
	}

    // write the frame down the stream
	frame.WriteFrame(func(bit bool) {
		driver.WriteBit(bit, writer)
	})


```

#### Receiving data

This is slightly more complicated, because we expect a lot of noise in the surrounding EM field.

```golang
func receive(bps int64, pinNum int) {
	pin, err := gpio.OpenPin(pinNum, gpio.ModeInput)
	if err != nil {
		panic(err)
	}

    // create the resulting frame
	frame := manchester.NewDataFrame()
	// create the driver
	driver := manchester.NewManchesterDriver(bps)

    // this structure is used to decouple the pin processing from the syscall watch.
	type Event struct {
		pinState  bool
		eventTime int64
	}

    // the receiver channel for the events from the pin
	events := make(chan Event, 1000)

	var latch sync.WaitGroup
	latch.Add(1)

    // the main loop to handle the events from the channel as they are reported by the watcher
	go func() {
		worker := func(evt Event) {
			var state manchester.Edge
			if evt.pinState {
				state = manchester.Up
			} else {
				state = manchester.Down
			}
			driver.ReadBit(state, evt.eventTime, func(x bool) {
				if frame.ReadBit(x) { // here it will return true if the frame was read completely.
					if frame.IsValid() {
						fmt.Println("============================")
						fmt.Println(string(frame.Data))
						latch.Done()
					} else {
					    // if the chechsum does not match - then the frame was corrupted on the way.
						fmt.Println("Missed")
						// reset the internal frame structures
						frame.Reset()
					}
				}
			})

		}

		for {
			worker(<-events)
		}
	}()

    // watch the events on the pin via syscall.
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
```