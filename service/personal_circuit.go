package service

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type PersonalCircuitStatus string

const (
	PersonalCircuitClosed   PersonalCircuitStatus = "closed"
	PersonalCircuitOpen     PersonalCircuitStatus = "open"
	PersonalCircuitHalfOpen PersonalCircuitStatus = "half_open"

	personalCircuitBaseBackoff       = 30 * time.Second
	personalCircuitMaxBackoff        = 15 * time.Minute
	personalCircuitModelBackoff      = 30 * time.Minute
	personalCircuitChannelBackoff    = 15 * time.Minute
	personalCircuitHalfOpenLease     = 30 * time.Second
	personalCircuitNotifySuppression = 10 * time.Minute
	personalCircuitMaxEntries        = 4096
	personalCircuitMaxNotifications  = 1024
	personalCircuitAllModels         = "*"
)

type PersonalCircuit struct {
	ChannelID           int                   `json:"channel_id"`
	ChannelName         string                `json:"channel_name,omitempty"`
	Model               string                `json:"model"`
	Scope               string                `json:"scope"`
	Status              PersonalCircuitStatus `json:"status"`
	ConsecutiveFailures int                   `json:"consecutive_failures"`
	OpenedAt            int64                 `json:"opened_at"`
	RetryAt             int64                 `json:"retry_at"`
	HalfOpenUntil       int64                 `json:"half_open_until,omitempty"`
	LastOutcome         RelayAttemptOutcome   `json:"last_outcome"`
	LastStatusCode      int                   `json:"last_status_code,omitempty"`
	LastErrorCode       string                `json:"last_error_code,omitempty"`
}

type PersonalCircuitTransition struct {
	ChannelID   int                   `json:"channel_id"`
	ChannelName string                `json:"channel_name,omitempty"`
	Model       string                `json:"model"`
	From        PersonalCircuitStatus `json:"from"`
	To          PersonalCircuitStatus `json:"to"`
	At          int64                 `json:"at"`
	Outcome     RelayAttemptOutcome   `json:"outcome,omitempty"`
	StatusCode  int                   `json:"status_code,omitempty"`
	ErrorCode   string                `json:"error_code,omitempty"`
	RetryAt     int64                 `json:"retry_at,omitempty"`
}

type PersonalCircuitPolicy struct {
	BaseBackoffSeconds    int64 `json:"base_backoff_seconds"`
	MaxBackoffSeconds     int64 `json:"max_backoff_seconds"`
	ModelBackoffSeconds   int64 `json:"model_backoff_seconds"`
	ChannelBackoffSeconds int64 `json:"channel_backoff_seconds"`
	HalfOpenLeaseSeconds  int64 `json:"half_open_lease_seconds"`
	Volatile              bool  `json:"volatile"`
}

type personalCircuitKey struct {
	channelID int
	model     string
}

type personalCircuitManager struct {
	mu          sync.Mutex
	now         func() time.Time
	entries     map[personalCircuitKey]*PersonalCircuit
	transitions []PersonalCircuitTransition
	notifiedAt  map[string]time.Time
}

func newPersonalCircuitManager(now func() time.Time) *personalCircuitManager {
	return &personalCircuitManager{
		now:        now,
		entries:    map[personalCircuitKey]*PersonalCircuit{},
		notifiedAt: map[string]time.Time{},
	}
}

var personalCircuits = newPersonalCircuitManager(time.Now)

func (m *personalCircuitManager) canAttempt(channelID int, modelName string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry := m.entryForAttemptLocked(channelID, modelName)
	if entry == nil {
		return true
	}
	now := m.now().Unix()
	switch entry.Status {
	case PersonalCircuitOpen:
		return now >= entry.RetryAt
	case PersonalCircuitHalfOpen:
		return now >= entry.HalfOpenUntil
	default:
		return true
	}
}

func (m *personalCircuitManager) claim(channelID int, modelName string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry := m.entryForAttemptLocked(channelID, modelName)
	if entry == nil {
		return true
	}
	now := m.now()
	if entry.Status == PersonalCircuitHalfOpen && now.Unix() < entry.HalfOpenUntil {
		return false
	}
	if entry.Status == PersonalCircuitOpen && now.Unix() < entry.RetryAt {
		return false
	}
	from := entry.Status
	entry.Status = PersonalCircuitHalfOpen
	entry.HalfOpenUntil = now.Add(personalCircuitHalfOpenLease).Unix()
	if from != PersonalCircuitHalfOpen {
		m.appendTransitionLocked(PersonalCircuitTransition{
			ChannelID: entry.ChannelID, ChannelName: entry.ChannelName, Model: entry.Model,
			From: from, To: PersonalCircuitHalfOpen, At: now.Unix(),
			Outcome: entry.LastOutcome, StatusCode: entry.LastStatusCode,
			ErrorCode: entry.LastErrorCode, RetryAt: entry.RetryAt,
		})
	}
	return true
}

func (m *personalCircuitManager) entryForAttemptLocked(channelID int, modelName string) *PersonalCircuit {
	if entry := m.entries[personalCircuitKey{channelID: channelID, model: personalCircuitAllModels}]; entry != nil {
		return entry
	}
	return m.entries[personalCircuitKey{channelID: channelID, model: modelName}]
}

func (m *personalCircuitManager) recordFailure(channelID int, channelName, modelName string, attempt RelayAttempt) *PersonalCircuitTransition {
	if !opensPersonalCircuit(attempt.Outcome) || channelID <= 0 || modelName == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	// circuitModel is derived solely from the failure type: AuthError and
	// ChannelUnavailable produce a channel-wide "*" entry; all other outcomes
	// are tracked per-model. The previous behaviour of promoting any failure to
	// channel-level whenever a "*" entry already existed caused unrelated
	// transient errors (e.g. 429 on a different model during a half-open probe)
	// to amplify the shared ConsecutiveFailures counter and push the backoff
	// toward the 15-minute ceiling far faster than any single model's true
	// error rate would warrant.
	circuitModel := personalCircuitModel(attempt.Outcome, modelName)
	key := personalCircuitKey{channelID: channelID, model: circuitModel}
	if circuitModel == personalCircuitAllModels {
		for existingKey := range m.entries {
			if existingKey.channelID == channelID && existingKey != key {
				delete(m.entries, existingKey)
			}
		}
	}
	entry := m.entries[key]
	from := PersonalCircuitClosed
	if entry == nil {
		m.pruneEntriesLocked()
		entry = &PersonalCircuit{ChannelID: channelID, Model: circuitModel, Scope: personalCircuitScope(circuitModel)}
		m.entries[key] = entry
	} else {
		from = entry.Status
	}
	entry.ChannelName = channelName
	entry.Status = PersonalCircuitOpen
	entry.ConsecutiveFailures++
	entry.OpenedAt = now.Unix()
	entry.HalfOpenUntil = 0
	entry.LastOutcome = attempt.Outcome
	entry.LastStatusCode = attempt.StatusCode
	entry.LastErrorCode = attempt.ErrorCode
	entry.RetryAt = now.Add(personalCircuitBackoff(attempt, entry.ConsecutiveFailures)).Unix()
	if from == PersonalCircuitOpen {
		return nil
	}
	transition := PersonalCircuitTransition{
		ChannelID: channelID, ChannelName: channelName, Model: circuitModel,
		From: from, To: PersonalCircuitOpen, At: now.Unix(), Outcome: attempt.Outcome,
		StatusCode: attempt.StatusCode, ErrorCode: attempt.ErrorCode, RetryAt: entry.RetryAt,
	}
	m.appendTransitionLocked(transition)
	return &transition
}

func (m *personalCircuitManager) recordSuccess(channelID int, modelName string) []PersonalCircuitTransition {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := []personalCircuitKey{
		{channelID: channelID, model: personalCircuitAllModels},
		{channelID: channelID, model: modelName},
	}
	transitions := make([]PersonalCircuitTransition, 0, len(keys))
	for _, key := range keys {
		entry := m.entries[key]
		if entry == nil {
			continue
		}
		delete(m.entries, key)
		transition := PersonalCircuitTransition{
			ChannelID: entry.ChannelID, ChannelName: entry.ChannelName, Model: entry.Model,
			From: entry.Status, To: PersonalCircuitClosed, At: m.now().Unix(),
		}
		m.appendTransitionLocked(transition)
		transitions = append(transitions, transition)
	}
	return transitions
}

func (m *personalCircuitManager) reset(channelIDs map[int]struct{}, modelName string) []PersonalCircuitTransition {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now().Unix()
	transitions := make([]PersonalCircuitTransition, 0)
	for key, entry := range m.entries {
		if _, ok := channelIDs[key.channelID]; !ok {
			continue
		}
		if modelName != "" && key.model != modelName && key.model != personalCircuitAllModels {
			continue
		}
		transition := PersonalCircuitTransition{
			ChannelID: entry.ChannelID, ChannelName: entry.ChannelName, Model: entry.Model,
			From: entry.Status, To: PersonalCircuitClosed, At: now,
		}
		delete(m.entries, key)
		m.appendTransitionLocked(transition)
		transitions = append(transitions, transition)
	}
	return transitions
}

func (m *personalCircuitManager) snapshot() ([]PersonalCircuit, []PersonalCircuitTransition) {
	m.mu.Lock()
	defer m.mu.Unlock()
	circuits := make([]PersonalCircuit, 0, len(m.entries))
	for _, entry := range m.entries {
		circuits = append(circuits, *entry)
	}
	sort.Slice(circuits, func(i, j int) bool {
		if circuits[i].RetryAt != circuits[j].RetryAt {
			return circuits[i].RetryAt < circuits[j].RetryAt
		}
		if circuits[i].ChannelID != circuits[j].ChannelID {
			return circuits[i].ChannelID < circuits[j].ChannelID
		}
		return circuits[i].Model < circuits[j].Model
	})
	transitions := append([]PersonalCircuitTransition(nil), m.transitions...)
	return circuits, transitions
}

func (m *personalCircuitManager) appendTransitionLocked(transition PersonalCircuitTransition) {
	m.transitions = append([]PersonalCircuitTransition{transition}, m.transitions...)
	if len(m.transitions) > 100 {
		m.transitions = m.transitions[:100]
	}
}

func (m *personalCircuitManager) shouldNotify(transition PersonalCircuitTransition) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	if len(m.notifiedAt) >= personalCircuitMaxNotifications {
		for key, notifiedAt := range m.notifiedAt {
			if now.Sub(notifiedAt) >= personalCircuitNotifySuppression {
				delete(m.notifiedAt, key)
			}
		}
	}
	key := fmt.Sprintf("%d|%s|%s", transition.ChannelID, transition.Model, transition.To)
	if notifiedAt, ok := m.notifiedAt[key]; ok && now.Sub(notifiedAt) < personalCircuitNotifySuppression {
		return false
	}
	m.notifiedAt[key] = now
	if len(m.notifiedAt) > personalCircuitMaxNotifications {
		var oldestKey string
		var oldest time.Time
		for candidateKey, notifiedAt := range m.notifiedAt {
			if oldestKey == "" || notifiedAt.Before(oldest) {
				oldestKey = candidateKey
				oldest = notifiedAt
			}
		}
		delete(m.notifiedAt, oldestKey)
	}
	return true
}

func (m *personalCircuitManager) pruneEntriesLocked() {
	if len(m.entries) < personalCircuitMaxEntries {
		return
	}
	var oldestKey personalCircuitKey
	oldestRetryAt := int64(0)
	found := false
	for key, entry := range m.entries {
		if !found || entry.RetryAt < oldestRetryAt {
			oldestKey = key
			oldestRetryAt = entry.RetryAt
			found = true
		}
	}
	if found {
		delete(m.entries, oldestKey)
	}
}

func opensPersonalCircuit(outcome RelayAttemptOutcome) bool {
	return outcome == RelayAttemptTransportError || outcome == RelayAttemptRateLimited ||
		outcome == RelayAttemptUpstream5xx || outcome == RelayAttemptModelUnavailable ||
		outcome == RelayAttemptAuthError || outcome == RelayAttemptChannelUnavailable
}

func personalCircuitModel(outcome RelayAttemptOutcome, modelName string) string {
	if outcome == RelayAttemptAuthError || outcome == RelayAttemptChannelUnavailable {
		return personalCircuitAllModels
	}
	return modelName
}

func personalCircuitScope(modelName string) string {
	if modelName == personalCircuitAllModels {
		return "channel"
	}
	return "model"
}

func personalCircuitBackoff(attempt RelayAttempt, failures int) time.Duration {
	if attempt.Outcome == RelayAttemptModelUnavailable {
		return personalCircuitModelBackoff
	}
	if attempt.Outcome == RelayAttemptAuthError || attempt.Outcome == RelayAttemptChannelUnavailable {
		return personalCircuitChannelBackoff
	}
	backoff := personalCircuitBaseBackoff
	for i := 1; i < failures && backoff < personalCircuitMaxBackoff; i++ {
		backoff *= 2
	}
	if backoff > personalCircuitMaxBackoff {
		backoff = personalCircuitMaxBackoff
	}
	if attempt.Outcome == RelayAttemptRateLimited && attempt.RetryAfterSeconds > 0 {
		retryAfter := time.Duration(attempt.RetryAfterSeconds) * time.Second
		if retryAfter > personalCircuitMaxBackoff {
			retryAfter = personalCircuitMaxBackoff
		}
		if retryAfter > backoff {
			backoff = retryAfter
		}
	}
	return backoff
}
