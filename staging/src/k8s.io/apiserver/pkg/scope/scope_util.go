package scope

// Expired returns true if the Scope can no longer be considered valid and any users of it must reset their state.
// This typically happens when a new scope generation has replaced an old one.
// If the ExpiredCh is not set, this function returns true (i.e. 'expired').
func Expired(s Scope) bool {
	select {
	case <-s.Expired():
		return true
	default:
		return false
	}
}
