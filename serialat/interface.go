package serialat

type ISerialAT interface {
	WriteLine(data []byte) (int, error)
	ReadLine() []byte
}
