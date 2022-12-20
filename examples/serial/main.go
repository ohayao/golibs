package main

import (
	"fmt"
	"time"

	"github.com/ohayao/golibs/serial"
)

func main() {
	s := serial.NewSerial("/dev/tty.usbserial-A907ENCL", 9600, 0x08, serial.ParityNone, serial.Stop1)
	if err := s.Open(); err != nil {
		panic(err)
	}
	defer func() {
		err := s.Close()
		fmt.Println(err)
	}()
	s.SetTailChars([]byte{0x0D, 0x0A})
	s.SetReadTimeout(time.Second * 5)
	go s.StartRecv()
	go func() {
		for {
			time.Sleep(time.Second * 5)
			cmd := fmt.Sprintf("AT+%02d?", time.Now().Second())
			s.Write([]byte(cmd))
			fmt.Printf("Send %s\n", cmd)
		}
	}()
	for {
		data := s.Parse([]byte{0x0D}, []byte{0x0D, 0x0A})
		if data != nil {
			fmt.Printf("Recv %s\n", string(data))
		}
	}
}
