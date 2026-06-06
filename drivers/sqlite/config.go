package sqlite

const DriverName = "sqlite"

type Config struct {
	// Path is the SQLite database path.
	//
	// Examples:
	//   Path: "bkv.db"
	//   Path: "/tmp/bkv.db"
	//   Path: ":memory:"
	//
	// If empty, ":memory:" is used.
	Path string

	// Prefix is optional.
	//
	// Example:
	//   Prefix: "myapp:"
	//
	// Key "session:123" becomes "myapp:session:123".
	Prefix string

	// Table is optional.
	//
	// If empty, "bkv" is used.
	Table string
}

func (Config) DriverName() string {
	return DriverName
}
