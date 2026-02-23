package postgres

import "fmt"

type Config struct {
	Host     string
	Port     int
	User     string `env:"DATABASE_USER" secret:"db-user"`
	Password string `env:"DATABASE_PASSWORD" secret:"db-password"`
	Database string
	Options  string
}

func (c Config) ConnectionString() string {
	return fmt.Sprintf(
		"user=%s password=%s host=%s port=%d dbname=%s %s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.Options,
	)
}
