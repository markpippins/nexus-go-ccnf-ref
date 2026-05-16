package snapshot

import "fmt"

type LockError string

const (
	ErrTriVersionLock LockError = "TRI_VERSION_LOCK_FAILURE"
)

func (e LockError) Error() string { return string(e) }

func ValidateTriVersionLock(s Snapshot) error {
	if s.CCNFVersion == 0 || s.CollapseVersion == 0 || s.RehydrationVersion == 0 {
		return fmt.Errorf("%w: zero version present (ccnf=%d, collapse=%d, rehydrate=%d)",
			ErrTriVersionLock, s.CCNFVersion, s.CollapseVersion, s.RehydrationVersion)
	}
	if s.CCNFVersion != s.CollapseVersion {
		return fmt.Errorf("%w: ccnf_version %d != collapse_version %d",
			ErrTriVersionLock, s.CCNFVersion, s.CollapseVersion)
	}
	if s.CCNFVersion != s.RehydrationVersion {
		return fmt.Errorf("%w: ccnf_version %d != rehydration_version %d",
			ErrTriVersionLock, s.CCNFVersion, s.RehydrationVersion)
	}
	return nil
}

func IsValidLock(s Snapshot) bool {
	return ValidateTriVersionLock(s) == nil
}
