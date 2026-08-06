package service

func (m *personalCircuitManager) staleAttemptLocked(key personalCircuitKey, attempt RelayAttempt) bool {
	latest, ok := m.latestAttempts[key]
	return ok && attempt.StartedAtMs > 0 && latest > 0 && attempt.StartedAtMs < latest
}

func (m *personalCircuitManager) markAttemptLocked(key personalCircuitKey, attempt RelayAttempt) {
	if attempt.StartedAtMs <= 0 || attempt.StartedAtMs <= m.latestAttempts[key] {
		return
	}
	m.latestAttempts[key] = attempt.StartedAtMs
	if len(m.latestAttempts) <= personalCircuitMaxEntries {
		return
	}
	var oldestKey personalCircuitKey
	var oldest int64
	for candidateKey, startedAt := range m.latestAttempts {
		if oldest == 0 || startedAt < oldest {
			oldestKey = candidateKey
			oldest = startedAt
		}
	}
	delete(m.latestAttempts, oldestKey)
}
