package main

import (
	"fmt"
	"time"

	"github.com/ohayao/golibs/serialat"
)

func main() {
	var isa serialat.ISerialAT
	at, err := serialat.NewSerialAT("/dev/tty.usbserial-A907ENCL", 9600, 0x08, serialat.ParityNone, serialat.Stop1)
	if err != nil {
		panic(err)
	}
	at.SetSuffix([]byte{0x0D, 0x0A})
	isa = at
	go func() {
		time.Sleep(time.Second * 10)
		//if setsuffix ，it will append "\r\n" at the tail
		isa.WriteLine([]byte("AT+COMMAND?"))

		time.Sleep(time.Second * 30)
		isa.WriteLine([]byte("AT+OTHERCOMMAND?"))
	}()
	for {
		resp := isa.ReadLine()
		if resp != nil {
			fmt.Printf("Recv %0X %s\n", resp, string(resp))
		}
	}
}
