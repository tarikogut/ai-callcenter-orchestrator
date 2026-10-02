package diameter

import (
	"fmt"
	"sync"
	"time"
)

// ActiveSession tracks reservation and accumulated usage for a call
type ActiveSession struct {
	SessionID      string
	SubscriptionID string
	RatingType     RatingType
	ReservedUnits  uint32
	TotalUsedUnits uint32
	TotalCost      float64
	RatePerUnit    float64
	StartTime      time.Time
	LastUpdateTime time.Time
}

// BalanceStore provides thread-safe in-memory balances and Diameter Ro rating engine
type BalanceStore struct {
	mu          sync.RWMutex
	balances    map[string]float64       // SubscriptionID -> balance amount
	sessions    map[string]*ActiveSession // SessionID -> ActiveSession
	defaultRate float64                  // default rate per unit (e.g. 0.50/min => ~0.00833/sec, or 0.10/sms)
}

// NewBalanceStore creates an initialized BalanceStore
func NewBalanceStore() *BalanceStore {
	return &BalanceStore{
		balances:    make(map[string]float64),
		sessions:    make(map[string]*ActiveSession),
		defaultRate: 0.01, // default 0.01 per unit
	}
}

// SetBalance sets credit balance for a subscriber / tenant
func (s *BalanceStore) SetBalance(subscriptionID string, balance float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.balances[subscriptionID] = balance
}

// GetBalance returns current balance for a subscriber
func (s *BalanceStore) GetBalance(subscriptionID string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.balances[subscriptionID]
}

// AddBalance adds funds to account
func (s *BalanceStore) AddBalance(subscriptionID string, amount float64) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.balances[subscriptionID] += amount
	return s.balances[subscriptionID]
}

// ProcessCCR handles Diameter Ro rating according to 3GPP TS 32.299
func (s *BalanceStore) ProcessCCR(req CCRRequest) (*CCAResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rate := req.RatePerUnit
	if rate <= 0 {
		rate = s.defaultRate
	}

	currentBalance, exists := s.balances[req.SubscriptionID]
	if !exists {
		// Auto-initialize balance with a default if not found
		currentBalance = 10.0
		s.balances[req.SubscriptionID] = currentBalance
	}

	resp := &CCAResponse{
		SessionID:       req.SessionID,
		RequestType:     req.RequestType,
		RequestNumber:   req.RequestNumber,
		ResultCode:      ResultCodeSuccess,
		ValidityTime:    120, // 2 minutes validity
		RemainingCredit: currentBalance,
	}

	switch req.RequestType {
	case CCRequestTypeInitial:
		// Check balance for requested quota
		requested := req.RequestedUnits
		if requested == 0 {
			requested = 60 // default 60s
		}

		neededCost := float64(requested) * rate
		if currentBalance < neededCost {
			// Grant partial units if any balance left
			if currentBalance > rate {
				granted := uint32(currentBalance / rate)
				s.balances[req.SubscriptionID] -= float64(granted) * rate
				s.sessions[req.SessionID] = &ActiveSession{
					SessionID:      req.SessionID,
					SubscriptionID: req.SubscriptionID,
					RatingType:     req.RatingType,
					ReservedUnits:  granted,
					TotalCost:      float64(granted) * rate,
					RatePerUnit:    rate,
					StartTime:      time.Now(),
					LastUpdateTime: time.Now(),
				}
				resp.GrantedUnits = granted
				resp.FinalUnitAction = 0 // TERMINATE when quota ends
				resp.RemainingCredit = s.balances[req.SubscriptionID]
				return resp, nil
			}

			// Credit limit reached
			resp.ResultCode = ResultCodeCreditLimitReached
			resp.GrantedUnits = 0
			resp.ErrorMessage = fmt.Sprintf("insufficient balance: have %.2f, need %.2f", currentBalance, neededCost)
			return resp, nil
		}

		// Reserve requested quota
		s.balances[req.SubscriptionID] -= neededCost
		s.sessions[req.SessionID] = &ActiveSession{
			SessionID:      req.SessionID,
			SubscriptionID: req.SubscriptionID,
			RatingType:     req.RatingType,
			ReservedUnits:  requested,
			TotalCost:      neededCost,
			RatePerUnit:    rate,
			StartTime:      time.Now(),
			LastUpdateTime: time.Now(),
		}
		resp.GrantedUnits = requested
		resp.RemainingCredit = s.balances[req.SubscriptionID]
		return resp, nil

	case CCRequestTypeUpdate:
		session, ok := s.sessions[req.SessionID]
		if !ok {
			// Session not found, create new
			session = &ActiveSession{
				SessionID:      req.SessionID,
				SubscriptionID: req.SubscriptionID,
				RatingType:     req.RatingType,
				RatePerUnit:    rate,
				StartTime:      time.Now(),
			}
			s.sessions[req.SessionID] = session
		}

		session.TotalUsedUnits += req.UsedUnits
		session.LastUpdateTime = time.Now()

		// Settle difference if used exceeded reservation, or reserve next chunk
		requested := req.RequestedUnits
		if requested == 0 {
			requested = 60
		}
		neededCost := float64(requested) * rate
		if currentBalance < neededCost {
			if currentBalance > rate {
				granted := uint32(currentBalance / rate)
				s.balances[req.SubscriptionID] -= float64(granted) * rate
				session.ReservedUnits = granted
				session.TotalCost += float64(granted) * rate
				resp.GrantedUnits = granted
				resp.FinalUnitAction = 0
				resp.RemainingCredit = s.balances[req.SubscriptionID]
				return resp, nil
			}
			resp.ResultCode = ResultCodeCreditLimitReached
			resp.GrantedUnits = 0
			resp.RemainingCredit = currentBalance
			return resp, nil
		}

		s.balances[req.SubscriptionID] -= neededCost
		session.ReservedUnits = requested
		session.TotalCost += neededCost
		resp.GrantedUnits = requested
		resp.RemainingCredit = s.balances[req.SubscriptionID]
		return resp, nil

	case CCRequestTypeTermination:
		session, ok := s.sessions[req.SessionID]
		if ok {
			// Reconcile unused reservation
			// If used < reserved, refund the difference
			if req.UsedUnits < session.ReservedUnits {
				unused := session.ReservedUnits - req.UsedUnits
				refund := float64(unused) * session.RatePerUnit
				s.balances[req.SubscriptionID] += refund
				session.TotalCost -= refund
			}
			session.TotalUsedUnits += req.UsedUnits
			delete(s.sessions, req.SessionID)
		}
		resp.GrantedUnits = 0
		resp.RemainingCredit = s.balances[req.SubscriptionID]
		return resp, nil

	case CCRequestTypeEvent:
		// Direct one-time debit (e.g. SMS)
		units := req.RequestedUnits
		if units == 0 {
			units = 1
		}
		cost := float64(units) * rate
		if currentBalance < cost {
			resp.ResultCode = ResultCodeCreditLimitReached
			resp.GrantedUnits = 0
			resp.RemainingCredit = currentBalance
			resp.ErrorMessage = fmt.Sprintf("insufficient balance for event: have %.2f, need %.2f", currentBalance, cost)
			return resp, nil
		}

		s.balances[req.SubscriptionID] -= cost
		resp.GrantedUnits = units
		resp.RemainingCredit = s.balances[req.SubscriptionID]
		return resp, nil

	default:
		resp.ResultCode = ResultCodeRatingFailed
		resp.ErrorMessage = "unsupported CC-Request-Type"
		return resp, nil
	}
}
