package db

// New initializes and returns a bun-backed Store for the given dbType and dsn.
// It is a small, non-breaking convenience wrapper around NewStoreFromDSN that
// also sets the package-level `store` used by the package helpers.
func New(dbType, dsn string) (Store, error) {
	s, err := NewStoreFromDSN(dbType, dsn)
	if err != nil {
		return nil, err
	}
	store = s
	return s, nil
}
