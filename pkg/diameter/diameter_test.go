package diameter

import (
	"context"
	"testing"
)

func TestDiameterPacketEncodeDecode(t *testing.T) {
	// Build a test message
	msg := &Message{
		Version:       DiameterVersion,
		Flags:         FlagRequest | FlagProxiable,
		CommandCode:   CmdCreditControl,
		ApplicationID: AppIDDiameterRo,
		HopByHopID:    0x12345678,
		EndToEndID:    0x9ABCDEF0,
	}

	sessionID := "call-session-001;123456"
	msg.AVPs = append(msg.AVPs,
		NewAVPString(AVPSessionID, AVPFlagMandatory, sessionID),
		NewAVPUint32(AVPCCRequestType, AVPFlagMandatory, CCRequestTypeInitial),
		NewAVPUint32(AVPCCRequestNumber, AVPFlagMandatory, 0),
		NewGroupedAVP(AVPSubscriptionID, AVPFlagMandatory,
			NewAVPUint32(AVPSubscriptionIDType, AVPFlagMandatory, SubscriptionTypeEndUserE164),
			NewAVPString(AVPSubscriptionIDData, AVPFlagMandatory, "905321112233"),
		),
	)

	encoded := msg.Encode()
	if len(encoded) < 20 {
		t.Fatalf("encoded message too short: %d bytes", len(encoded))
	}

	decoded, err := DecodeMessage(encoded)
	if err != nil {
		t.Fatalf("failed to decode message: %v", err)
	}

	if decoded.CommandCode != CmdCreditControl {
		t.Errorf("expected command code %d, got %d", CmdCreditControl, decoded.CommandCode)
	}
	if decoded.HopByHopID != 0x12345678 {
		t.Errorf("expected hop-by-hop ID 0x12345678, got 0x%X", decoded.HopByHopID)
	}
	if decoded.EndToEndID != 0x9ABCDEF0 {
		t.Errorf("expected end-to-end ID 0x9ABCDEF0, got 0x%X", decoded.EndToEndID)
	}

	sidAVP := decoded.FindAVP(AVPSessionID)
	if sidAVP == nil {
		t.Fatalf("Session-Id AVP not found")
	}
	if string(sidAVP.Data) != sessionID {
		t.Errorf("expected Session-Id %q, got %q", sessionID, string(sidAVP.Data))
	}

	subAVP := decoded.FindAVP(AVPSubscriptionID)
	if subAVP == nil {
		t.Fatalf("Subscription-Id AVP not found")
	}
	subChildren, err := DecodeGroupedAVP(subAVP)
	if err != nil {
		t.Fatalf("failed to decode grouped subscription AVP: %v", err)
	}
	if len(subChildren) != 2 {
		t.Errorf("expected 2 children in subscription ID, got %d", len(subChildren))
	}
}

func TestDiameterClientAndBalanceStore(t *testing.T) {
	client := NewClient(ClientConfig{
		OriginHost:  "test.orchestrator.local",
		OriginRealm: "orchestrator.local",
	})
	store := client.GetBalanceStore()
	tenantID := "tenant_eczane_hayat"

	// 1. Initial balance setup
	store.SetBalance(tenantID, 5.0) // 5.0 TL
	bal := store.GetBalance(tenantID)
	if bal != 5.0 {
		t.Fatalf("expected balance 5.0, got %f", bal)
	}

	ctx := context.Background()

	// 2. CCR-Initial: reserve 60 seconds at rate 0.01/sec = 0.60 TL
	initReq := CCRRequest{
		SessionID:      "sess-call-101",
		SubscriptionID: tenantID,
		RatingType:     RatingTypeVoiceTime,
		RequestedUnits: 60,
		RatePerUnit:    0.01,
	}

	initResp, err := client.SendCCRInitial(ctx, initReq)
	if err != nil {
		t.Fatalf("CCR-Initial failed: %v", err)
	}
	if initResp.ResultCode != ResultCodeSuccess {
		t.Fatalf("expected ResultCodeSuccess, got %d: %s", initResp.ResultCode, initResp.ErrorMessage)
	}
	if initResp.GrantedUnits != 60 {
		t.Errorf("expected 60 granted units, got %d", initResp.GrantedUnits)
	}
	if store.GetBalance(tenantID) != 4.40 {
		t.Errorf("expected remaining balance 4.40, got %.2f", store.GetBalance(tenantID))
	}

	// 3. CCR-Update: 60 seconds used, reserve next 60 seconds (cost 0.60 TL)
	updReq := CCRRequest{
		SessionID:      "sess-call-101",
		SubscriptionID: tenantID,
		RequestNumber:  1,
		RatingType:     RatingTypeVoiceTime,
		UsedUnits:      60,
		RequestedUnits: 60,
		RatePerUnit:    0.01,
	}

	updResp, err := client.SendCCRUpdate(ctx, updReq)
	if err != nil {
		t.Fatalf("CCR-Update failed: %v", err)
	}
	if updResp.ResultCode != ResultCodeSuccess {
		t.Fatalf("expected ResultCodeSuccess in update, got %d", updResp.ResultCode)
	}
	approxEqual := func(a, b float64) bool {
		diff := a - b
		if diff < 0 {
			diff = -diff
		}
		return diff < 0.0001
	}

	if !approxEqual(store.GetBalance(tenantID), 3.80) {
		t.Errorf("expected balance 3.80, got %.2f", store.GetBalance(tenantID))
	}

	// 4. CCR-Terminate: only used 30 seconds of the 60 reserved, refund 30 seconds (0.30 TL)
	termReq := CCRRequest{
		SessionID:      "sess-call-101",
		SubscriptionID: tenantID,
		RequestNumber:  2,
		RatingType:     RatingTypeVoiceTime,
		UsedUnits:      30,
		RatePerUnit:    0.01,
	}

	termResp, err := client.SendCCRTerminate(ctx, termReq)
	if err != nil {
		t.Fatalf("CCR-Terminate failed: %v", err)
	}
	if termResp.ResultCode != ResultCodeSuccess {
		t.Fatalf("expected ResultCodeSuccess in terminate, got %d", termResp.ResultCode)
	}
	// 3.80 + 0.30 refund = 4.10
	if !approxEqual(store.GetBalance(tenantID), 4.10) {
		t.Errorf("expected balance 4.10 after refund, got %.2f", store.GetBalance(tenantID))
	}

	// 5. CCR-Event (SMS rating test): Cost 0.10 TL
	smsReq := CCRRequest{
		SessionID:      "sess-sms-202",
		SubscriptionID: tenantID,
		RatingType:     RatingTypeSmsEvent,
		RequestedUnits: 1,
		RatePerUnit:    0.10,
	}

	smsResp, err := client.SendCCREvent(ctx, smsReq)
	if err != nil {
		t.Fatalf("CCR-Event failed: %v", err)
	}
	if smsResp.ResultCode != ResultCodeSuccess {
		t.Fatalf("expected ResultCodeSuccess for SMS event, got %d", smsResp.ResultCode)
	}
	if !approxEqual(store.GetBalance(tenantID), 4.00) {
		t.Errorf("expected balance 4.00 after SMS, got %.2f", store.GetBalance(tenantID))
	}

	// 6. Test Credit Limit Reached
	store.SetBalance(tenantID, 0.05)
	failSmsReq := CCRRequest{
		SessionID:      "sess-sms-203",
		SubscriptionID: tenantID,
		RatingType:     RatingTypeSmsEvent,
		RequestedUnits: 1,
		RatePerUnit:    0.10,
	}
	failResp, err := client.SendCCREvent(ctx, failSmsReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failResp.ResultCode != ResultCodeCreditLimitReached {
		t.Errorf("expected CreditLimitReached (4012), got %d", failResp.ResultCode)
	}
}
