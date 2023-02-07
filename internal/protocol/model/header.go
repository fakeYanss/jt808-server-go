package model

import (
	"encoding/binary"

	"github.com/pkg/errors"

	"github.com/fakeYanss/jt808-server-go/internal/util"
)

var (
	ErrDecodeHeader = errors.New("Decode header error")
)

// 定义消息头
type MsgHeader struct {
	MsgID uint16 `json:"msgID"` // 消息ID
	MsgBodyAttr
	ProtocolVersion  uint8  `json:"protocolVersion"` // 协议版本号，默认0表示2011/2013版本，其他为2019后续版本，每次修订递增，初始为1
	PhoneNumber      string `json:"phoneNumber"`     // 终端手机号
	SerialNumber     uint16 `json:"serialNumber"`    // 消息流水号
	MsgFragmentation        // 消息包封装项

	Idx int32 `json:"-"` // 读取的packet header下标ID
}

// 将[]byte解码成消息头结构体
func (h *MsgHeader) Decode(pkt []byte) error {
	var idx int32

	h.MsgID = binary.BigEndian.Uint16(pkt[:idx+2]) // 消息id [0,2)位
	idx += 2

	err := h.MsgBodyAttr.Decode(pkt[idx : idx+2]) // 消息体属性 [2,4)位
	if err != nil {
		return ErrDecodeHeader
	}
	idx += 2

	if h.VersionSign {
		h.ProtocolVersion = pkt[idx] // 2019版本，协议版本号 第4位
		idx++
	}

	// 2013版本，phoneNumber [5,11)位 长度6位；2019版本，phoneNumber [5,15)位 长度10位。
	if h.VersionSign {
		h.PhoneNumber = util.BCD2NumberStr(pkt[idx : idx+10])
		idx += 10
	} else {
		h.PhoneNumber = util.BCD2NumberStr(pkt[idx : idx+6])
		idx += 6
	}

	h.SerialNumber = binary.BigEndian.Uint16(pkt[idx : idx+2])
	idx += 2

	if h.PacketFragmented {
		err = h.MsgFragmentation.Decode(pkt[idx : idx+2]) // 消息包封装项，两位
		if err != nil {
			return ErrDecodeHeader
		}
		idx += 2
	}

	h.Idx = idx

	return nil
}

// 将消息头结构体编码成[]byte
func (h *MsgHeader) Encode() ([]byte, error) {
	pkt := make([]byte, 0)

	id := make([]byte, 2)
	binary.BigEndian.PutUint16(id, h.MsgID) // 消息id
	pkt = append(pkt, id...)

	bodyAttrPkt, err := h.MsgBodyAttr.Encode() // 消息体属性
	if err != nil {
		return nil, err
	}
	pkt = append(pkt, bodyAttrPkt...)

	pkt = append(pkt, h.ProtocolVersion) // 协议版本号

	pkt = append(pkt, util.NumberStr2BCD(h.PhoneNumber)...) // 手机号

	sn := make([]byte, 2)
	binary.BigEndian.PutUint16(sn, h.SerialNumber) // 消息流水号
	pkt = append(pkt, sn...)

	fragPkt, err := h.MsgFragmentation.Encode() // 消息包封装项
	if err != nil {
		return nil, err
	}
	pkt = append(pkt, fragPkt...)

	return pkt, nil
}

// 消息体属性字段的bit位
const (
	bodyLengthBit    uint16 = 0b0000001111111111
	encryptionBit    uint16 = 0b0001110000000000
	fragmentationBit uint16 = 0b0010000000000000
	versionSignBit   uint16 = 0b0100000000000000
	extraBit         uint16 = 0b1000000000000000
)

// 加密类型
const (
	EncryptionNone    = "Encryption None"
	EncryptionRSA     = "Encryption RSA"
	EncryptionUnknown = "Encryption Unknown"
)

// 定义消息体属性
type MsgBodyAttr struct {
	BodyLength       uint16 `json:"bodyLength"`       // 消息体长度
	Encryption       string `json:"encryption"`       // 是否加密
	PacketFragmented bool   `json:"packetFragmented"` // 是否分包
	VersionSign      bool   `json:"versionSign"`      // 版本标识，false表示协议版本是最早一期的版本，true表示已经引入协议版本标识的功能; 对应到消息头解析有差别
	Extra            uint8  `json:"extra"`            // 预留一个bit位的保留字段

	encryptionOriginal uint8 // 加密方式原文, 回响应时用到
}

func (a *MsgBodyAttr) Decode(sub []byte) error {
	// 2-3位字节转为二进制数字
	bitNum := binary.BigEndian.Uint16(sub)

	a.BodyLength = bitNum & bodyLengthBit // 消息体长度 低10位

	// 加密方式 10-12位
	a.encryptionOriginal = uint8((bitNum & encryptionBit) >> 10)
	switch a.encryptionOriginal {
	case 0b000:
		a.Encryption = EncryptionNone
	case 0b001:
		a.Encryption = EncryptionRSA
	default:
		a.Encryption = EncryptionUnknown
	}

	a.PacketFragmented = (bitNum&fragmentationBit>>13 == 1) // 分包 13位

	a.VersionSign = (bitNum&versionSignBit>>14 == 1) // 版本标识 14位
	a.Extra = uint8(bitNum & extraBit >> 15)         // 保留 15位
	return nil
}

func (a *MsgBodyAttr) Encode() ([]byte, error) {
	var bitNum uint16

	bitNum += a.BodyLength                       // 消息体长度
	bitNum += uint16(a.encryptionOriginal) << 10 // 加密方式
	if a.PacketFragmented {
		bitNum += 1 << 13 // 分包
	}
	if a.VersionSign {
		bitNum += 1 << 14 // 版本标识
	}
	bitNum += uint16(a.Extra) << 15 // 保留位

	pkt := make([]byte, 2)
	binary.BigEndian.PutUint16(pkt, bitNum)
	return pkt, nil
}

// 定义分包的封装项
type MsgFragmentation struct {
	Total uint16 `json:"total"` // 分包后的包总数
	Index uint16 `json:"index"` // 包序号，从1开始
}

func (f *MsgFragmentation) Decode(sub []byte) error {
	f.Total = binary.BigEndian.Uint16(sub[:2])
	f.Index = binary.BigEndian.Uint16(sub[2:])
	return nil
}

func (f *MsgFragmentation) Encode() ([]byte, error) {
	pkt := make([]byte, 0)

	tot := make([]byte, 2)
	binary.BigEndian.PutUint16(tot, f.Total)
	pkt = append(pkt, tot...)

	idx := make([]byte, 2)
	binary.BigEndian.PutUint16(idx, f.Index)
	pkt = append(pkt, idx...)

	return pkt, nil
}
