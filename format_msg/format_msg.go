package format_msg

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func FromBytes16(b []byte) (uint16, error) {
	buf := bytes.NewReader(b)
	var result uint16
	err := binary.Read(buf, binary.BigEndian, &result)
	return result, err
}

func FromBytes8(b []byte) (uint8, error) {
	buf := bytes.NewReader(b)
	var result uint8
	err := binary.Read(buf, binary.BigEndian, &result)
	return result, err
}

func From8ToBytes(i uint8) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, i)
	return buf.Bytes(), err
}

func From16ToBytes(i uint16) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, i)
	return buf.Bytes(), err
}

func PackingMsg(i uint8, j uint16) []byte {
	command, err := From8ToBytes(i)
	if err != nil {
		fmt.Println("Packing error uint8")
	}
	segment, err := From16ToBytes(j)
	if err != nil {
		fmt.Println("Packing error uint16")
	}
	return append(command, append(segment, []byte("\n")...)...)
}

func UnpackingMsg(msg []byte) (uint8, uint16) {
	ui8, err := FromBytes8(msg[:1])
	if err != nil {
		fmt.Println("Upacking error uint8")
	}
	ui16, err := FromBytes16(msg[1:])
	if err != nil {
		fmt.Println("Upacking error uint16", msg[1:])
	}
	return ui8, ui16
}
