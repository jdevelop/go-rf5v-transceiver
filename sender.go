package main

import (
	"github.com/jdevelop/go-manchester/manchester"
	"github.com/davecheney/gpio"
	"fmt"
	"flag"
)

func main() {

	senderPin := flag.Int("sp", 21, "Sender pin")
	//receiverPin := *flag.Int("rp", 2, "Receiver pin")
	bps := flag.Int64("bps", 3, "BPS")
	text := flag.String("text", "AA", "sample text")

	flag.Parse()

	fmt.Printf("Using PIN %1d and bps speed %2d\n", *senderPin, *bps)

	driver := manchester.NewManchesterDriver(*bps)
	frame := manchester.BuildDataFrame([]byte(*text))

	pin, _ := gpio.OpenPin(*senderPin, gpio.ModeOutput)

	defer pin.Close()

	writer := func(level bool) {
		if level {
			pin.Set()
		} else {
			pin.Clear()
		}
	}

	for i := 1; i < 20; i++ {
		pin.Set()
		driver.Sleep()
		pin.Clear()
		driver.Sleep()
	}

	frame.WriteFrame(func(bit bool) {
		driver.WriteBit(bit, writer)
	})

	fmt.Println()
	fmt.Println("Done")

}
