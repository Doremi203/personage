package postgres

import "fmt"

type Config struct {
	Host     string
	Port     int
	User     string
	Password string `json:"-"`
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
