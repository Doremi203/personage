package sqs

type Config struct {
	Endpoint  string
	AccessKey string `json:"-"`
	SecretKey string `json:"-"`
	Region    string
	QueueURL  string
}
