package model

import (
	"encoding/binary"
)

type JT808Cmd interface {
	GenCmd(JT808Msg) error
	Encode() ([]byte, error) // struct -> []byte
}

type ResultCode uint8

const (
	ResultSuccess      ResultCode = 0
	ResultFail         ResultCode = 1
	ResultErrMsg       ResultCode = 2
	ResultNotSupported ResultCode = 3
)

// 平台通用应答
type Cmd8001 struct {
	Header             *MsgHeader `json:"header"`
	AnswerSerialNumber uint16     `json:"answerSerialNumber"` // 应答流水号，对应平台消息的流水号
	AnswerMessageID    uint16     `json:"answerMessageId"`    // 应答ID，对应平台消息的ID
	Result             ResultCode `json:"result"`             // 结果，0成功/确认，1失败，2消息有误，3不支持
}

func (c *Cmd8001) GenCmd(msg JT808Msg) error {
	header := msg.GetHeader()
	c.AnswerSerialNumber = header.SerialNumber
	c.AnswerMessageID = header.MsgID
	c.Result = 0

	c.Header = header
	c.Header.MsgID = 0x8001

	return nil
}

func (c *Cmd8001) Encode() (pkt []byte, err error) {
	asn := make([]byte, 2)
	binary.BigEndian.PutUint16(asn, c.AnswerSerialNumber)
	pkt = append(pkt, asn...)

	amid := make([]byte, 2)
	binary.BigEndian.PutUint16(amid, c.AnswerMessageID)
	pkt = append(pkt, amid...)

	pkt = append(pkt, byte(c.Result))

	c.Header.Attr.BodyLength = uint16(len(pkt))

	headerPkt, err := c.Header.Encode()
	if err != nil {
		return nil, err
	}

	pkt = append(headerPkt, pkt...)

	return
}

type CmdResult byte

const (
	ResSuccess               CmdResult = 0
	ResCarAlreadyRegister    CmdResult = 1
	ResCarNotExist           CmdResult = 2
	ResDeviceAlreadyRegister CmdResult = 3
	ResDeviceNotExist        CmdResult = 4
)

// 终端注册应答消息
type Cmd8100 struct {
	Header             *MsgHeader `json:"header"`
	AnswerSerialNumber uint16     `json:"answerSerialNumber"` // 应答流水号，对应平台消息的流水号
	Result             CmdResult  `json:"result"`             // 结果，0成功，1车辆已被注册，2数据库中无此车辆，3此终端已被注册，4数据库中无此终端
	AuthCode           string     `json:"authCode"`           // 鉴权码
}

func (c *Cmd8100) GenCmd(msg JT808Msg) error {
	m := msg.(*Msg0100)
	c.AnswerSerialNumber = m.Header.SerialNumber
	c.Result = 0
	c.AuthCode = "AuthCode" // 初始值，在后续处理中根据id重写

	c.Header = m.Header
	c.Header.MsgID = 0x8100

	return nil
}

func (c *Cmd8100) Encode() (pkt []byte, err error) {
	asn := make([]byte, 2)
	binary.BigEndian.PutUint16(asn, c.AnswerSerialNumber)
	pkt = append(pkt, asn...)

	pkt = append(pkt, byte(c.Result))

	pkt = append(pkt, []byte(c.AuthCode)...)

	c.Header.Attr.BodyLength = uint16(len(pkt))

	headerPkt, err := c.Header.Encode()
	if err != nil {
		return nil, err
	}

	pkt = append(headerPkt, pkt...)

	return
}
