package local

const DriverName = "local"

type Config struct{}

func (Config) DriverName() string {
	return DriverName
}
