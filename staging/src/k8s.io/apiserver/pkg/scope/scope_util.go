package scope

import (
	scopesv1alpha1 "k8s.io/api/scopes/v1alpha1"
)

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

// GetServerScopeVersion returns a ServerScopeVersion with the provided APIServerID and StoreID if it exists.
func GetServerScopeVersion(status scopesv1alpha1.ScopeStatus, apiServerID, storeID string) *scopesv1alpha1.ServerScopeVersion {
	for _, c := range status.ServerScopeVersions {
		if c.APIServerID == apiServerID && c.StoreID == storeID {
			return &c
		}
	}
	return nil
}

// SetServerScopeVersion adds/replaces the given ServerScopeVersion in the Scope status.
// If the ServerScopeVersion that we are about to add already exists, we will update it.
func SetServerScopeVersion(status *scopesv1alpha1.ScopeStatus, ssv scopesv1alpha1.ServerScopeVersion) {
	currentCond := GetServerScopeVersion(*status, ssv.APIServerID, ssv.StoreID)
	if currentCond != nil {
		status.ServerScopeVersions = filterOutServerScopeVersion(status.ServerScopeVersions, ssv.APIServerID, ssv.StoreID)
	}
	status.ServerScopeVersions = append(status.ServerScopeVersions, ssv)
}

// RemoveServerScopeVersion removes the ssv with the provided type from the replicaset status.
func RemoveServerScopeVersion(status *scopesv1alpha1.ScopeStatus, apiServerID, storeID string) {
	status.ServerScopeVersions = filterOutServerScopeVersion(status.ServerScopeVersions, apiServerID, storeID)
}

// filterOutServerScopeVersion returns a new slice of server scope versions without the provided (apiServerID, storeID)
func filterOutServerScopeVersion(conditions []scopesv1alpha1.ServerScopeVersion, apiServerID, storeID string) []scopesv1alpha1.ServerScopeVersion {
	var newServerScopeVersions []scopesv1alpha1.ServerScopeVersion
	for _, c := range conditions {
		if c.APIServerID == apiServerID && c.StoreID == storeID {
			continue
		}
		newServerScopeVersions = append(newServerScopeVersions, c)
	}
	return newServerScopeVersions
}
