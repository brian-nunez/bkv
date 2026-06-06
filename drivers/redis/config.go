package redis

const DriverName = "redis"

type Config struct {
	Secure   bool
	Username string
	Password string
	Addr     string
	DB       int
	Prefix   string
}

func (Config) DriverName() string {
	return DriverName
}
