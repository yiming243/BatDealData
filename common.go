package main

import (
	"bytes"
	"encoding/asn1"
	"encoding/binary"
	"errors"
	"math/big"
)

func Int16ToBytes(i int16) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint16(buf, uint16(i))
	return buf
}

func BytesToInt16(buf []byte) int16 {
	return int16(binary.BigEndian.Uint16(buf))
}

func Int32ToBytes(i int32) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf, uint32(i))
	return buf
}

func BytesToInt32(buf []byte) int32 {
	return int32(binary.BigEndian.Uint32(buf))
}

func SignDerData2RS(signDerData []byte) ([]byte, error){
	type Sm2SignT struct{
		X *big.Int
		Y *big.Int
	}
	var signObj Sm2SignT

	_, err := asn1.Unmarshal(signDerData, &signObj)
	if err != nil {
		return nil, errors.New("Unmarshal fail")
	}

	rs := make([]byte, 64)
	copy(rs[32 - len(signObj.X.Bytes()):32], signObj.X.Bytes())
	copy(rs[(64 - len(signObj.Y.Bytes())):64], signObj.Y.Bytes())

	return rs, nil
}

func BytesCombine(pBytes ...[]byte) []byte {
	len := len(pBytes)
	s := make([][]byte, len)
	for index := 0; index < len; index++ {
		s[index] = pBytes[index]
	}
	return bytes.Join(s, []byte{})
}