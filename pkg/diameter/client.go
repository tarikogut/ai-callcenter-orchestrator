package diameter

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// RatingType represents voice duration or event (SMS) rating
type RatingType string

const (
	RatingTypeVoiceTime RatingType = "VOICE_TIME" // Rating in seconds
	RatingTypeSmsEvent  RatingType = "SMS_EVENT"  // Rating per event
)

// CCRRequest contains high level fields needed to assemble a CCR
type CCRRequest struct {
	SessionID         string
	RequestType       uint32 // CCRequestTypeInitial, CCRequestTypeUpdate, CCRequestTypeTermination, CCRequestTypeEvent
	RequestNumber     uint32
	SubscriptionID    string // e.g. MSISDN, Account ID
	RatingType        RatingType
	UsedUnits         uint32  // seconds for voice, 1 for sms
	RequestedUnits    uint32  // seconds for voice reserve
	RatePerUnit       float64 // optional local fallback rate
	ServiceIdentifier uint32  // optional service identifier
}

// CCAResponse contains parsed result from CCA
type CCAResponse struct {
	SessionID       string
	RequestType     uint32
	RequestNumber   uint32
	ResultCode      uint32
	GrantedUnits    uint32
	ValidityTime    uint32
	FinalUnitAction uint32
	RemainingCredit float64
	ErrorMessage    string
}

// ClientConfig holds configuration for Diameter Ro client
type ClientConfig struct {
	OriginHost       string
	OriginRealm      string
	DestinationHost  string
	DestinationRealm string
	ServerAddr       string        // "host:port", e.g. "127.0.0.1:3868"
	Timeout          time.Duration // Network timeout
	EnableMockServer bool          // If true and ServerAddr fails/empty, runs mock fallback
}

// RoClient interface defines operations for Diameter Ro Credit-Control
type RoClient interface {
	SendCCRInitial(ctx context.Context, req CCRRequest) (*CCAResponse, error)
	SendCCRUpdate(ctx context.Context, req CCRRequest) (*CCAResponse, error)
	SendCCRTerminate(ctx context.Context, req CCRRequest) (*CCAResponse, error)
	SendCCREvent(ctx context.Context, req CCRRequest) (*CCAResponse, error)
	Close() error

	// Balance Store access
	GetBalanceStore() *BalanceStore
}

// Client implements RoClient
type Client struct {
	config       ClientConfig
	hopByHopSeed uint32
	endToEndSeed uint32
	balanceStore *BalanceStore
	mu           sync.Mutex
	conn         net.Conn
}

// NewClient creates a new Diameter Ro client
func NewClient(cfg ClientConfig) *Client {
	if cfg.OriginHost == "" {
		cfg.OriginHost = "orchestrator.icell.cloud"
	}
	if cfg.OriginRealm == "" {
		cfg.OriginRealm = "icell.cloud"
	}
	if cfg.DestinationRealm == "" {
		cfg.DestinationRealm = "icell.cloud"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 3 * time.Second
	}

	return &Client{
		config:       cfg,
		hopByHopSeed: 1000,
		endToEndSeed: 2000,
		balanceStore: NewBalanceStore(),
	}
}

func (c *Client) GetBalanceStore() *BalanceStore {
	return c.balanceStore
}

func (c *Client) nextHopByHop() uint32 {
	return atomic.AddUint32(&c.hopByHopSeed, 1)
}

func (c *Client) nextEndToEnd() uint32 {
	return atomic.AddUint32(&c.endToEndSeed, 1)
}

// BuildCCR constructs a RFC 4006 CCR Message
func (c *Client) BuildCCR(req CCRRequest) *Message {
	msg := &Message{
		Version:       DiameterVersion,
		Flags:         FlagRequest | FlagProxiable,
		CommandCode:   CmdCreditControl,
		ApplicationID: AppIDDiameterRo,
		HopByHopID:    c.nextHopByHop(),
		EndToEndID:    c.nextEndToEnd(),
	}

	msg.AVPs = append(msg.AVPs,
		NewAVPString(AVPSessionID, AVPFlagMandatory, req.SessionID),
		NewAVPString(AVPOriginHost, AVPFlagMandatory, c.config.OriginHost),
		NewAVPString(AVPOriginRealm, AVPFlagMandatory, c.config.OriginRealm),
		NewAVPString(AVPDestinationRealm, AVPFlagMandatory, c.config.DestinationRealm),
		NewAVPUint32(AVPAuthApplicationID, AVPFlagMandatory, AppIDDiameterRo),
		NewAVPUint32(AVPCCRequestType, AVPFlagMandatory, req.RequestType),
		NewAVPUint32(AVPCCRequestNumber, AVPFlagMandatory, req.RequestNumber),
	)

	if c.config.DestinationHost != "" {
		msg.AVPs = append(msg.AVPs, NewAVPString(AVPDestinationHost, AVPFlagMandatory, c.config.DestinationHost))
	}

	// Subscription-Id grouped AVP
	if req.SubscriptionID != "" {
		subType := NewAVPUint32(AVPSubscriptionIDType, AVPFlagMandatory, SubscriptionTypeEndUserE164)
		subData := NewAVPString(AVPSubscriptionIDData, AVPFlagMandatory, req.SubscriptionID)
		msg.AVPs = append(msg.AVPs, NewGroupedAVP(AVPSubscriptionID, AVPFlagMandatory, subType, subData))
	}

	// Requested-Service-Unit if requesting quota
	if req.RequestedUnits > 0 {
		var inner []AVP
		if req.RatingType == RatingTypeVoiceTime {
			inner = append(inner, NewAVPUint32(AVPCCTime, AVPFlagMandatory, req.RequestedUnits))
		} else {
			inner = append(inner, NewAVPUint32(AVPCCServiceSpecificUnits, AVPFlagMandatory, req.RequestedUnits))
		}
		msg.AVPs = append(msg.AVPs, NewGroupedAVP(AVPRequestedServiceUnit, AVPFlagMandatory, inner...))
	}

	// Used-Service-Unit if reporting used quota
	if req.UsedUnits > 0 {
		var inner []AVP
		if req.RatingType == RatingTypeVoiceTime {
			inner = append(inner, NewAVPUint32(AVPCCTime, AVPFlagMandatory, req.UsedUnits))
		} else {
			inner = append(inner, NewAVPUint32(AVPCCServiceSpecificUnits, AVPFlagMandatory, req.UsedUnits))
		}
		msg.AVPs = append(msg.AVPs, NewGroupedAVP(AVPUsedServiceUnit, AVPFlagMandatory, inner...))
	}

	return msg
}

// ParseCCA extracts CCAResponse from a CCA Message
func ParseCCA(msg *Message) (*CCAResponse, error) {
	if msg == nil {
		return nil, errors.New("nil diameter message")
	}

	resp := &CCAResponse{
		ResultCode: ResultCodeSuccess,
	}

	if sidAVP := msg.FindAVP(AVPSessionID); sidAVP != nil {
		resp.SessionID = string(sidAVP.Data)
	}

	if rcAVP := msg.FindAVP(AVPResultCode); rcAVP != nil && len(rcAVP.Data) >= 4 {
		resp.ResultCode = binary.BigEndian.Uint32(rcAVP.Data[:4])
	}

	if reqTypeAVP := msg.FindAVP(AVPCCRequestType); reqTypeAVP != nil && len(reqTypeAVP.Data) >= 4 {
		resp.RequestType = binary.BigEndian.Uint32(reqTypeAVP.Data[:4])
	}

	if reqNumAVP := msg.FindAVP(AVPCCRequestNumber); reqNumAVP != nil && len(reqNumAVP.Data) >= 4 {
		resp.RequestNumber = binary.BigEndian.Uint32(reqNumAVP.Data[:4])
	}

	if gsuAVP := msg.FindAVP(AVPGrantedServiceUnit); gsuAVP != nil {
		children, err := DecodeGroupedAVP(gsuAVP)
		if err == nil {
			for _, child := range children {
				if (child.Code == AVPCCTime || child.Code == AVPCCServiceSpecificUnits) && len(child.Data) >= 4 {
					resp.GrantedUnits = binary.BigEndian.Uint32(child.Data[:4])
				}
			}
		}
	}

	if vtAVP := msg.FindAVP(AVPValidityTime); vtAVP != nil && len(vtAVP.Data) >= 4 {
		resp.ValidityTime = binary.BigEndian.Uint32(vtAVP.Data[:4])
	}

	if fuaAVP := msg.FindAVP(AVPFinalUnitIndication); fuaAVP != nil {
		children, err := DecodeGroupedAVP(fuaAVP)
		if err == nil {
			for _, child := range children {
				if child.Code == AVPFinalUnitAction && len(child.Data) >= 4 {
					resp.FinalUnitAction = binary.BigEndian.Uint32(child.Data[:4])
				}
			}
		}
	}

	return resp, nil
}

// SendCCRInitial sends CCR-Initial (Reserve quota for starting call)
func (c *Client) SendCCRInitial(ctx context.Context, req CCRRequest) (*CCAResponse, error) {
	req.RequestType = CCRequestTypeInitial
	return c.sendOrFallback(ctx, req)
}

// SendCCRUpdate sends CCR-Update (Debit consumed quota and reserve next chunk)
func (c *Client) SendCCRUpdate(ctx context.Context, req CCRRequest) (*CCAResponse, error) {
	req.RequestType = CCRequestTypeUpdate
	return c.sendOrFallback(ctx, req)
}

// SendCCRTerminate sends CCR-Terminate (Final debited usage and close session)
func (c *Client) SendCCRTerminate(ctx context.Context, req CCRRequest) (*CCAResponse, error) {
	req.RequestType = CCRequestTypeTermination
	return c.sendOrFallback(ctx, req)
}

// SendCCREvent sends CCR-Event (One-shot rating, e.g. SMS)
func (c *Client) SendCCREvent(ctx context.Context, req CCRRequest) (*CCAResponse, error) {
	req.RequestType = CCRequestTypeEvent
	return c.sendOrFallback(ctx, req)
}

func (c *Client) sendOrFallback(ctx context.Context, req CCRRequest) (*CCAResponse, error) {
	// If a live Diameter server is configured, try sending via network
	if c.config.ServerAddr != "" {
		cca, err := c.sendOverNetwork(ctx, req)
		if err == nil {
			return cca, nil
		}
		// If network error and MockServer is disabled, return error
		if !c.config.EnableMockServer {
			return nil, fmt.Errorf("diameter network error: %w", err)
		}
	}

	// In-memory rating & balance store fallback
	return c.balanceStore.ProcessCCR(req)
}

func (c *Client) sendOverNetwork(ctx context.Context, req CCRRequest) (*CCAResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := net.DialTimeout("tcp", c.config.ServerAddr, c.config.Timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ccrMsg := c.BuildCCR(req)
	data := ccrMsg.Encode()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(c.config.Timeout))
	}

	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("write error: %w", err)
	}

	// Read response header (20 bytes)
	header := make([]byte, 20)
	if _, err := conn.Read(header); err != nil {
		return nil, fmt.Errorf("read header error: %w", err)
	}
	msgLen := int(uint32(header[1])<<16 | uint32(header[2])<<8 | uint32(header[3]))
	if msgLen < 20 || msgLen > 65535 {
		return nil, fmt.Errorf("invalid response message length %d", msgLen)
	}

	buf := make([]byte, msgLen)
	copy(buf[:20], header)
	if _, err := conn.Read(buf[20:]); err != nil {
		return nil, fmt.Errorf("read body error: %w", err)
	}

	ccaMsg, err := DecodeMessage(buf)
	if err != nil {
		return nil, fmt.Errorf("decode CCA error: %w", err)
	}

	return ParseCCA(ccaMsg)
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}
