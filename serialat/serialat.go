package serialat

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/xluohome/serial"
)

type (
	Parity   byte
	StopBits byte
)

const (
	Stop1     StopBits = 1
	Stop1Half StopBits = 15
	Stop2     StopBits = 2

	ParityNone  Parity = Parity(serial.ParityNone)
	ParityOdd   Parity = Parity(serial.ParityOdd)
	ParityEven  Parity = Parity(serial.ParityEven)
	ParityMark  Parity = Parity(serial.ParityMark)  // parity bit is always 1
	ParitySpace Parity = Parity(serial.ParitySpace) // parity bit is always 0
)

type SerialAT struct {
	port     *serial.Port
	suffix   []byte   //用于发送命令时自动加在尾部
	data     []byte   //接收到的待解析的数据
	cmds     [][]byte //解析好的命令
	dataLock sync.Mutex
	cmdsLock sync.Mutex
}

// 初始化AT指令串口设备
func NewSerialAT(device string, baudRate int, size byte, parity Parity, stop StopBits) (*SerialAT, error) {
	var cfg = &serial.Config{
		Name:   device,
		Baud:   baudRate,
		Size:   size,
		Parity: serial.Parity(parity),
	}
	port, err := serial.OpenPort(cfg)
	if err != nil {
		return nil, err
	}
	_at := &SerialAT{
		port:   port,
		suffix: make([]byte, 0),
	}
	go _at.recv()
	return _at, nil
}

// 接收数据
func (that *SerialAT) recv() {
	read := make([]byte, 2048)
	for {
		length, err := that.port.Read(read)
		if err == io.EOF || (err == nil && length == 0) {
			continue
		} else if err != nil {
			fmt.Println(err)
		} else {
			that.dataLock.Lock()
			that.data = append(that.data, read[:length]...)
			that.dataLock.Unlock()
		}
	}
}

// 添加命令
func (that *SerialAT) cmd_push(cmd []byte) {
	that.cmdsLock.Lock()
	that.cmds = append(that.cmds, cmd)
	that.cmdsLock.Unlock()
}

// 解析数据
func (that *SerialAT) parse() {
	for {
		that.removePrefixCRLF()
		//找到第一个"|r"标志
		crIndex := bytes.Index(that.data, []byte{0x0D})
		if crIndex > 0 {
			cmd := that.data[:crIndex]
			that.cmd_push(cmd)
			that.data = that.data[crIndex:]
		} else {
			break
		}
	}
}

// 清除prefix的CRLF
func (that *SerialAT) removePrefixCRLF() {
	for {
		if len(that.data) > 0 {
			if that.data[0] == '\r' || that.data[0] == '\n' {
				that.data = that.data[1:]
				continue
			} else {
				break
			}
		} else {
			break
		}
	}
}

// 设置命令尾部字符
func (that *SerialAT) SetSuffix(suffix []byte) {
	that.suffix = suffix
}

// 向串口写入命令
func (that *SerialAT) WriteLine(cmd []byte) (int, error) {
	if that.port == nil {
		return 0, errors.New("serial device empty")
	}
	_cmd := append(cmd, that.suffix...)
	return that.port.Write(_cmd)
}

// 读取一行数据
func (that *SerialAT) ReadLine() []byte {
	that.dataLock.Lock()
	that.parse()
	that.dataLock.Unlock()
	that.cmdsLock.Lock()
	defer that.cmdsLock.Unlock()
	if len(that.cmds) < 1 {
		return nil
	}
	cmd := that.cmds[0]
	that.cmds = that.cmds[1:]
	return cmd
}
