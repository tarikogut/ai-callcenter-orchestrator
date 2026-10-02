package diameter

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// RFC 6733 / RFC 4006 Constants
const (
	DiameterVersion = 1

	// Application IDs
	AppIDDiameterBase = 0
	AppIDDiameterRo   = 4 // Credit-Control Application (RFC 4006 / 3GPP TS 32.299)

	// Command Codes
	CmdCreditControl = 272 // CCR / CCA

	// Command Flags
	FlagRequest       = 0x80
	FlagProxiable     = 0x40
	FlagError         = 0x20
	FlagRetransmitted = 0x10

	// AVP Flags
	AVPFlagMandatory = 0x40
	AVPFlagVendor    = 0x80

	// Standard AVPs (RFC 6733 / RFC 4006 / 3GPP TS 32.299)
	AVPUserName                   = 1
	AVPClass                      = 25
	AVPSessionID                  = 263
	AVPOriginHost                 = 264
	AVPOriginRealm                = 296
	AVPDestinationRealm           = 283
	AVPDestinationHost            = 293
	AVPResultCode                 = 268
	AVPExperimentalResult         = 297
	AVPAuthApplicationID          = 258
	AVPCCRequestType              = 416
	AVPCCRequestNumber            = 415
	AVPSubscriptionID             = 443
	AVPSubscriptionIDType         = 450
	AVPSubscriptionIDData         = 444
	AVPRequestedServiceUnit       = 437
	AVPGrantedServiceUnit         = 431
	AVPUsedServiceUnit            = 446
	AVPCCCorrelationID            = 411
	AVPMultipleServicesIndicator  = 455
	AVPMultipleServicesCC         = 456
	AVPServiceIdentifier          = 439
	AVPRatingGroup                = 432
	AVPCCUnitType                 = 454
	AVPUnitValue                  = 445
	AVPValueDigits                = 447
	AVPExponent                   = 429
	AVPCurrencyCode               = 425
	AVPCCMoney                    = 413
	AVPCCTotalOctets              = 412
	AVPCCInputOctets              = 414
	AVPCCOutputOctets             = 417
	AVPCCTime                     = 420
	AVPCCServiceSpecificUnits     = 417
	AVPValidityTime               = 448
	AVPFinalUnitIndication        = 430
	AVPFinalUnitAction            = 449
	AVPServiceContextID           = 461

	// CC-Request-Type values (RFC 4006 Section 8.3)
	CCRequestTypeInitial     uint32 = 1
	CCRequestTypeUpdate      uint32 = 2
	CCRequestTypeTermination uint32 = 3
	CCRequestTypeEvent       uint32 = 4

	// Result-Code values (RFC 6733 / RFC 4006)
	ResultCodeSuccess             uint32 = 2001
	ResultCodeLimitedSuccess      uint32 = 2002
	ResultCodeEndUserServiceDenied uint32 = 4010
	ResultCodeCreditLimitReached   uint32 = 4012
	ResultCodeRatingFailed        uint32 = 5031
	ResultCodeUserUnknown         uint32 = 5030

	// Subscription-Id-Type (RFC 4006 Section 8.47)
	SubscriptionTypeEndUserE164 uint32 = 0
	SubscriptionTypeEndUserIMSI uint32 = 1
	SubscriptionTypeEndUserSIP  uint32 = 2
	SubscriptionTypeEndUserNAI  uint32 = 3
	SubscriptionTypeEndUserPrivate uint32 = 4
)

// AVP represents a Diameter Attribute-Value Pair.
type AVP struct {
	Code     uint32
	Flags    uint8
	VendorID uint32
	Data     []byte
}

// Encode serializes the AVP into byte slice with proper padding (multiple of 4 bytes).
func (a *AVP) Encode() []byte {
	hasVendor := (a.Flags & AVPFlagVendor) != 0
	headerLen := 8
	if hasVendor {
		headerLen = 12
	}
	dataLen := len(a.Data)
	totalLen := headerLen + dataLen

	// Pad to 4-byte boundary
	padLen := (4 - (dataLen % 4)) % 4
	buf := make([]byte, totalLen+padLen)

	binary.BigEndian.PutUint32(buf[0:4], a.Code)
	buf[4] = a.Flags
	// 3-byte length
	buf[5] = byte((totalLen >> 16) & 0xFF)
	buf[6] = byte((totalLen >> 8) & 0xFF)
	buf[7] = byte(totalLen & 0xFF)

	offset := 8
	if hasVendor {
		binary.BigEndian.PutUint32(buf[8:12], a.VendorID)
		offset = 12
	}

	copy(buf[offset:], a.Data)
	// padding bytes are zeros by default
	return buf
}

// DecodeAVP decodes a single AVP from byte slice, returning the AVP, total bytes read including padding, and error.
func DecodeAVP(data []byte) (*AVP, int, error) {
	if len(data) < 8 {
		return nil, 0, errors.New("data too short for AVP header")
	}

	code := binary.BigEndian.Uint32(data[0:4])
	flags := data[4]
	length := int(uint32(data[5])<<16 | uint32(data[6])<<8 | uint32(data[7]))

	hasVendor := (flags & AVPFlagVendor) != 0
	minHeaderLen := 8
	if hasVendor {
		minHeaderLen = 12
	}

	if length < minHeaderLen {
		return nil, 0, fmt.Errorf("invalid AVP length %d (min %d)", length, minHeaderLen)
	}
	if len(data) < length {
		return nil, 0, fmt.Errorf("data buffer %d bytes shorter than AVP length %d", len(data), length)
	}

	var vendorID uint32
	offset := 8
	if hasVendor {
		vendorID = binary.BigEndian.Uint32(data[8:12])
		offset = 12
	}

	avpData := make([]byte, length-offset)
	copy(avpData, data[offset:length])

	padLen := (4 - (length % 4)) % 4
	totalRead := length + padLen
	if totalRead > len(data) {
		totalRead = len(data)
	}

	return &AVP{
		Code:     code,
		Flags:    flags,
		VendorID: vendorID,
		Data:     avpData,
	}, totalRead, nil
}

// Message represents a Diameter packet.
type Message struct {
	Version       uint8
	Flags         uint8
	CommandCode   uint32
	ApplicationID uint32
	HopByHopID    uint32
	EndToEndID    uint32
	AVPs          []AVP
}

// Encode serializes the Diameter Message.
func (m *Message) Encode() []byte {
	var avpBytes []byte
	for _, avp := range m.AVPs {
		avpBytes = append(avpBytes, avp.Encode()...)
	}

	totalLen := 20 + len(avpBytes)
	buf := make([]byte, totalLen)

	buf[0] = m.Version
	buf[1] = byte((totalLen >> 16) & 0xFF)
	buf[2] = byte((totalLen >> 8) & 0xFF)
	buf[3] = byte(totalLen & 0xFF)

	buf[4] = m.Flags
	buf[5] = byte((m.CommandCode >> 16) & 0xFF)
	buf[6] = byte((m.CommandCode >> 8) & 0xFF)
	buf[7] = byte(m.CommandCode & 0xFF)

	binary.BigEndian.PutUint32(buf[8:12], m.ApplicationID)
	binary.BigEndian.PutUint32(buf[12:16], m.HopByHopID)
	binary.BigEndian.PutUint32(buf[16:20], m.EndToEndID)

	copy(buf[20:], avpBytes)
	return buf
}

// DecodeMessage deserializes bytes into a Diameter Message.
func DecodeMessage(data []byte) (*Message, error) {
	if len(data) < 20 {
		return nil, errors.New("diameter packet too short (<20 bytes)")
	}

	version := data[0]
	msgLen := int(uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3]))
	if msgLen < 20 || msgLen > len(data) {
		return nil, fmt.Errorf("invalid diameter message length: %d (buf: %d)", msgLen, len(data))
	}

	flags := data[4]
	cmdCode := uint32(data[5])<<16 | uint32(data[6])<<8 | uint32(data[7])
	appID := binary.BigEndian.Uint32(data[8:12])
	hbh := binary.BigEndian.Uint32(data[12:16])
	ete := binary.BigEndian.Uint32(data[16:20])

	msg := &Message{
		Version:       version,
		Flags:         flags,
		CommandCode:   cmdCode,
		ApplicationID: appID,
		HopByHopID:    hbh,
		EndToEndID:    ete,
	}

	offset := 20
	for offset < msgLen {
		avp, readBytes, err := DecodeAVP(data[offset:msgLen])
		if err != nil {
			return nil, fmt.Errorf("error decoding AVP at offset %d: %w", offset, err)
		}
		msg.AVPs = append(msg.AVPs, *avp)
		offset += readBytes
	}

	return msg, nil
}

// FindAVP finds the first AVP matching the code.
func (m *Message) FindAVP(code uint32) *AVP {
	for i := range m.AVPs {
		if m.AVPs[i].Code == code {
			return &m.AVPs[i]
		}
	}
	return nil
}

// FindAllAVPs finds all AVPs matching the code.
func (m *Message) FindAllAVPs(code uint32) []AVP {
	var list []AVP
	for _, a := range m.AVPs {
		if a.Code == code {
			list = append(list, a)
		}
	}
	return list
}

// Helper functions for creating AVPs
func NewAVP(code uint32, flags uint8, data []byte) AVP {
	return AVP{
		Code:  code,
		Flags: flags,
		Data:  data,
	}
}

func NewAVPUint32(code uint32, flags uint8, val uint32) AVP {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, val)
	return NewAVP(code, flags, b)
}

func NewAVPUint64(code uint32, flags uint8, val uint64) AVP {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, val)
	return NewAVP(code, flags, b)
}

func NewAVPString(code uint32, flags uint8, str string) AVP {
	return NewAVP(code, flags, []byte(str))
}

func NewGroupedAVP(code uint32, flags uint8, innerAVPs ...AVP) AVP {
	var buf []byte
	for _, a := range innerAVPs {
		buf = append(buf, a.Encode()...)
	}
	return NewAVP(code, flags, buf)
}

// DecodeGroupedAVP decodes child AVPs inside a grouped AVP's Data
func DecodeGroupedAVP(avp *AVP) ([]AVP, error) {
	var children []AVP
	offset := 0
	for offset < len(avp.Data) {
		child, readBytes, err := DecodeAVP(avp.Data[offset:])
		if err != nil {
			return nil, err
		}
		children = append(children, *child)
		offset += readBytes
	}
	return children, nil
}
