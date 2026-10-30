package app

// Version is the app version. It must match `version` in build/config.yml (a test checks this).
const Version = "1.17.2"

// Version tells the screen which version is running.
func (s *Store) Version() string { return Version }
