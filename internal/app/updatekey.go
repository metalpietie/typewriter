package app

// updatePublicKey is the public half of the key that signs the releases (see tools/release and docs/releasing.md).
// The app installs an update only when its signature matches this key. It is made by "go run ./tools/release keygen".
const updatePublicKey = "3r2gQDzZsH/+JfaDrTCdBDfTtL4MN25hjkXiREiZf5o="
