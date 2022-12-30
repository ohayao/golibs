package serial

import (
	"bytes"
	"io"
	"sync"
	"time"

	"github.com/xluohome/serial"
	another "go.bug.st/serial"
)

type (
	Parity   byte
	StopBits byte
	Serial   struct {
		config      *serial.Config
		port        *serial.Port
		readTimeout time.Duration
		cacheLength int
		readLength  int
		suffix      []byte
		cacheData   []byte
		cacheLock   sync.Mutex
	}
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

// 初始化串口配置
// 默认缓存大小2048，读取缓冲大小1024
func NewSerial(device string, baudrate int, size byte, parity Parity, stopbits StopBits) *Serial {
	cfg := &serial.Config{
		Name:     device,
		Baud:     baudrate,
		Size:     size,
		StopBits: serial.StopBits(stopbits),
	}
	s := &Serial{
		config:      cfg,
		readTimeout: time.Duration(0),
		cacheLength: 2048,
		readLength:  1024,
		cacheData:   make([]byte, 0),
	}
	return s
}

// 获取串口列表
func GetPortsList() ([]string, error) {
	return another.GetPortsList()
}

// 设置读取超时，以防止读取一半数据终端造成数据不完整
// 在超时时间内，未接收到结束标志，则清空缓存数据
func (that *Serial) SetReadTimeout(timeout time.Duration) *Serial {
	that.readTimeout = timeout
	return that
}

// 设置数据缓存大小
func (that *Serial) SetCacheLength(length int) *Serial {
	that.cacheLength = length
	return that
}

// 设置读取串口数据大小
func (that *Serial) SetReadLength(length int) *Serial {
	that.readLength = length
	return that
}

// 设置尾部追加字符串
// eg. \r\n ==> 0x0D 0x0A
func (that *Serial) SetTailChars(tail []byte) *Serial {
	that.suffix = tail
	return that
}

// 打开串口
func (that *Serial) Open() error {
	port, err := serial.OpenPort(that.config)
	if err != nil {
		return err
	}
	that.port = port
	return nil
}

// 关闭串口
func (that *Serial) Close() error {
	return that.port.Close()
}

// 向串口写入数据
// 如果设置了尾部字符，则会自动添加
func (that *Serial) Write(data []byte) (int, error) {
	if that.suffix != nil {
		data = append(data, that.suffix...)
	}
	return that.port.Write(data)
}

// 开始从串口中接收数据
func (that *Serial) StartRecv() error {
	//读取之前清空缓冲区数据
	that.port.Flush()
	readbuf := make([]byte, that.readLength)
	last := time.Now()
	var res error
	for {
		length, err := that.port.Read(readbuf)
		if err == io.EOF || (err == nil && length == 0) {
			continue
		} else if err != nil {
			res = err
			goto exit
		} else {
			cur := time.Now()
			diff := cur.Sub(last)
			last = cur
			that.cacheLock.Lock()
			if (that.readTimeout > 0 && diff > that.readTimeout) || len(that.cacheData) > that.cacheLength {
				that.cacheData = make([]byte, that.cacheLength)
				that.cacheData = readbuf[:length]
			} else {
				that.cacheData = append(that.cacheData, readbuf[:length]...)
			}
			that.cacheLock.Unlock()
		}
	}
exit:
	that.cacheData = make([]byte, 0)
	return res
}

// 解析数据
// flag 解析的标志字符串 eg. \r\n 0x0D 0x0A
// 解析完后，后删除已解析的数据，并删除（逐一删除）行首标志字符
func (that *Serial) Parse(flag []byte, remove []byte) []byte {
	that.cacheLock.Lock()
	defer that.cacheLock.Unlock()
	for {
		isMatch := false
		for _, b := range remove {
			if len(that.cacheData) > 0 && that.cacheData[0] == b {
				that.cacheData = that.cacheData[1:]
				isMatch = true
			}
		}
		if isMatch {
			continue
		} else {
			break
		}
	}
	index := bytes.Index(that.cacheData, flag)
	if index > 0 {
		res := that.cacheData[:index]
		that.cacheData = that.cacheData[index:]
		return res
	}
	return nil
}
