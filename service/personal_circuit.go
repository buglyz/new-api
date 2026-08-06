package service

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/relaykit/types"
)

type PersonalCircuitStatus string

const (
	PersonalCircuitClosed   PersonalCircuitStatus = "closed"
	PersonalCircuitOpen     PersonalCircuitStatus = "open"
	PersonalCircuitHalfOpen PersonalCircuitStatus = "half_open"

	personalCircuitBaseBackoff       = 15 * time.Second
	personalCircuitMaxBackoff        = 5 * time.Minute
	personalCircuitModelBackoff      = 10 * time.Minute
	personalCircuitAuthBackoff       = 15 * time.Minute
	personalCircuitChannelBackoff    = 10 * time.Minute
	personalCircuitHalfOpenLease     = 2 * time.Minute
	personalCircuitFailureThreshold  = 3
	personalCircuitNotifySuppression = 10 * time.Minute
	personalCircuitMaxEntries        = 4096
	personalCircuitMaxNotifications  = 1024
	personalCircuitAllModels         = "*"
)

type PersonalCircuit struct {
	ChannelID             int                   `json:"channel_id"`
	ChannelName           string                `json:"channel_name,omitempty"`
	Model                 string                `json:"model"`
	Scope                 string                `json:"scope"`
	Status                PersonalCircuitStatus `json:"status"`
	ConsecutiveFailures   int                   `json:"consecutive_failures"`
	OpenedAt              int64                 `json:"opened_at"`
	RetryAt               int64                 `json:"retry_at"`
	HalfOpenUntil         int64                 `json:"half_open_until,omitempty"`
	LastOutcome           RelayAttemptOutcome   `json:"last_outcome"`
	LastStatusCode        int                   `json:"last_status_code,omitempty"`
	LastErrorCode         string                `json:"last_error_code,omitempty"`
	halfOpenProbeInFlight bool
	halfOpenProbeStarted  int64
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
	FailureThreshold      int   `json:"failure_threshold"`
	BaseBackoffSeconds    int64 `json:"base_backoff_seconds"`
	MaxBackoffSeconds     int64 `json:"max_backoff_seconds"`
	ModelBackoffSeconds   int64 `json:"model_backoff_seconds"`
	AuthBackoffSeconds    int64 `json:"auth_backoff_seconds"`
	ChannelBackoffSeconds int64 `json:"channel_backoff_seconds"`
	HalfOpenLeaseSeconds  int64 `json:"half_open_lease_seconds"`
	Volatile              bool  `json:"volatile"`
}

type personalCircuitKey struct {
	channelID int
	model     string
}

type personalCircuitManager struct {
	mu             sync.Mutex
	now            func() time.Time
	entries        map[personalCircuitKey]*PersonalCircuit
	latestAttempts map[personalCircuitKey]int64
	transitions    []PersonalCircuitTransition
	notifiedAt     map[string]time.Time
}

func newPersonalCircuitManager(now func() time.Time) *personalCircuitManager {
	return &personalCircuitManager{
		now:            now,
		entries:        map[personalCircuitKey]*PersonalCircuit{},
		latestAttempts: map[personalCircuitKey]int64{},
		notifiedAt:     map[string]time.Time{},
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
		if !entry.halfOpenProbeInFlight {
			return true
		}
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
	if entry.Status == PersonalCircuitClosed {
		return true
	}
	if entry.Status == PersonalCircuitHalfOpen && entry.halfOpenProbeInFlight && now.Unix() < entry.HalfOpenUntil {
		return false
	}
	if entry.Status == PersonalCircuitOpen && now.Unix() < entry.RetryAt {
		return false
	}
	from := entry.Status
	entry.Status = PersonalCircuitHalfOpen
	entry.HalfOpenUntil = now.Add(personalCircuitHalfOpenLease).Unix()
	entry.halfOpenProbeInFlight = true
	entry.halfOpenProbeStarted = now.UnixMilli()
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
	channelEntry := m.entries[personalCircuitKey{channelID: channelID, model: personalCircuitAllModels}]
	if channelEntry != nil && channelEntry.Status != PersonalCircuitClosed {
		return channelEntry
	}
	if modelEntry := m.entries[personalCircuitKey{channelID: channelID, model: modelName}]; modelEntry != nil {
		return modelEntry
	}
	return channelEntry
}

func (m *personalCircuitManager) recordFailure(channelID int, channelName, modelName string, attempt RelayAttempt) *PersonalCircuitTransition {
	if !opensPersonalCircuit(attempt.Outcome) || channelID <= 0 || modelName == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	// Keep model-specific configuration errors from taking unrelated models out
	// of rotation. Authentication and genuinely channel-wide failures still use
	// the shared "*" entry.
	circuitModel := personalCircuitModel(attempt, modelName)
	key := personalCircuitKey{channelID: channelID, model: circuitModel}
	if m.staleAttemptLocked(key, attempt) || staleHalfOpenProbe(m.entryForAttemptLocked(channelID, modelName), attempt) {
		return nil
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
	m.markAttemptLocked(key, attempt)
	entry.ChannelName = channelName
	entry.ConsecutiveFailures++
	entry.LastOutcome = attempt.Outcome
	entry.LastStatusCode = attempt.StatusCode
	entry.LastErrorCode = attempt.ErrorCode
	entry.halfOpenProbeInFlight = false
	entry.halfOpenProbeStarted = 0
	if from == PersonalCircuitHalfOpen || entry.ConsecutiveFailures >= personalCircuitFailureThreshold {
		entry.Status = PersonalCircuitOpen
		entry.OpenedAt = now.Unix()
		entry.HalfOpenUntil = 0
		backoffFailures := entry.ConsecutiveFailures - personalCircuitFailureThreshold + 1
		if from == PersonalCircuitHalfOpen && backoffFailures < 2 {
			backoffFailures = 2
		}
		entry.RetryAt = now.Add(personalCircuitBackoff(attempt, backoffFailures)).Unix()
	} else {
		entry.Status = PersonalCircuitClosed
		entry.OpenedAt = 0
		entry.HalfOpenUntil = 0
		entry.RetryAt = 0
		return nil
	}
	if circuitModel == personalCircuitAllModels {
		for existingKey := range m.entries {
			if existingKey.channelID == channelID && existingKey != key {
				m.markAttemptLocked(existingKey, attempt)
				delete(m.entries, existingKey)
			}
		}
	}
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

func (m *personalCircuitManager) recordSuccess(channelID int, modelName string, attempt ...RelayAttempt) []PersonalCircuitTransition {
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
		if len(attempt) > 0 && (m.staleAttemptLocked(key, attempt[0]) || staleHalfOpenProbe(entry, attempt[0])) {
			continue
		}
		if len(attempt) > 0 {
			m.markAttemptLocked(key, attempt[0])
		}
		delete(m.entries, key)
		if entry.Status == PersonalCircuitClosed {
			continue
		}
		transition := PersonalCircuitTransition{
			ChannelID: entry.ChannelID, ChannelName: entry.ChannelName, Model: entry.Model,
			From: entry.Status, To: PersonalCircuitClosed, At: m.now().Unix(),
		}
		m.appendTransitionLocked(transition)
		transitions = append(transitions, transition)
	}
	return transitions
}

func staleHalfOpenProbe(entry *PersonalCircuit, attempt RelayAttempt) bool {
	return entry != nil && entry.halfOpenProbeInFlight && attempt.StartedAtMs > 0 &&
		entry.halfOpenProbeStarted > 0 && attempt.StartedAtMs < entry.halfOpenProbeStarted
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
		delete(m.entries, key)
		if entry.Status == PersonalCircuitClosed {
			continue
		}
		transition := PersonalCircuitTransition{
			ChannelID: entry.ChannelID, ChannelName: entry.ChannelName, Model: entry.Model,
			From: entry.Status, To: PersonalCircuitClosed, At: now,
		}
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
		if entry.Status == PersonalCircuitClosed {
			continue
		}
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

func personalCircuitModel(attempt RelayAttempt, modelName string) string {
	if attempt.Outcome == RelayAttemptAuthError ||
		(attempt.Outcome == RelayAttemptChannelUnavailable && !isModelScopedChannelError(attempt)) {
		return personalCircuitAllModels
	}
	return modelName
}

func isModelScopedChannelError(attempt RelayAttempt) bool {
	switch types.ErrorCode(attempt.ErrorCode) {
	case types.ErrorCodeChannelParamOverrideInvalid,
		types.ErrorCodeChannelHeaderOverrideInvalid,
		types.ErrorCodeChannelModelMappedError:
		return true
	default:
		return false
	}
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
	if attempt.Outcome == RelayAttemptAuthError {
		return personalCircuitAuthBackoff
	}
	if attempt.Outcome == RelayAttemptChannelUnavailable && !isModelScopedChannelError(attempt) {
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
