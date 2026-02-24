package webapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/log"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/spf13/viper"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
)

type grpcConfig struct {
	Port int
}

type httpConfig struct {
	Port int
}

type loggingConfig struct {
	Level  string
	Format string
}

type swaggerUIConfig struct {
	Path    string
	Enabled bool
}

type Config struct {
	grpc      grpcConfig
	http      httpConfig
	logging   loggingConfig
	swaggerUI swaggerUIConfig

	viperLoader  *viper.Viper
	ycSDKClient  *ycsdk.SDK
	logger       log.Logger
	secretsCache map[string]*lockbox.Payload // Cache for secrets by "id:version"
}

func (c *Config) ReadSection(ctx context.Context, name string, cfg any) error {
	err := c.readSection(ctx, name, cfg)
	if err != nil {
		return err
	}
	c.logger.Infof("loaded custom config %v", errors.Token("section", name))

	return nil
}

func (c *Config) readSection(ctx context.Context, name string, cfg any) error {
	err := c.viperLoader.UnmarshalKey(name, cfg)
	if err != nil {
		return errors.WrapFailf(err, "read section %v", errors.Token("name", name))
	}

	err = c.processValues(ctx, cfg)
	if err != nil {
		return errors.WrapFailf(err, "process values for config with %v", errors.Token("name", name))
	}

	return nil
}

func (c *Config) processValues(ctx context.Context, cfg any) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.Errorf("want pointer to struct, got %T", cfg)
	}
	v = v.Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)

		if !fv.CanSet() {
			continue
		}

		if fv.Kind() != reflect.String {
			continue
		}

		strValue := fv.String()
		if strValue == "" {
			continue
		}

		var finalValue string
		var err error

		switch {
		case strings.HasPrefix(strValue, "env:"):
			envName := strings.TrimPrefix(strValue, "env:")
			finalValue = os.Getenv(envName)
			if finalValue == "" {
				return errors.Errorf("environment variable not found: %v", errors.Token("env", envName))
			}
		case strings.HasPrefix(strValue, "secret:"):
			// Parse secret:{id}:{version}:{key}
			finalValue, err = c.loadSecretValue(ctx, strValue)
			if err != nil {
				return errors.WrapFailf(err, "load secret for field %q", field.Name)
			}
		default:
			// Use the YAML value as-is
			continue
		}

		fv.SetString(finalValue)
	}

	return nil
}

func (c *Config) loadSecretValue(ctx context.Context, secretSpec string) (string, error) {
	// Parse secret:{id}:{version}:{key}
	parts := strings.SplitN(secretSpec, ":", 4)
	if len(parts) != 4 || parts[0] != "secret" {
		return "", errors.Errorf(
			"invalid secret format, expected secret:{id}:{version}:{key}, got %v",
			errors.Token("spec", secretSpec),
		)
	}

	secretID := parts[1]
	versionID := parts[2]
	key := parts[3]

	if c.ycSDKClient == nil {
		return "", errors.Error("yandex cloud SDK client not initialized")
	}

	cacheKey := secretID + ":" + versionID

	var secret *lockbox.Payload
	if cachedSecret, exists := c.secretsCache[cacheKey]; exists {
		secret = cachedSecret
	} else {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		fetchedSecret, err := c.ycSDKClient.LockboxPayload().Payload().Get(ctx, &lockbox.GetPayloadRequest{
			SecretId:  secretID,
			VersionId: versionID,
		})
		if err != nil {
			return "", errors.WrapFailf(
				err,
				"get secret from lockbox %v %v",
				errors.Token("id", secretID),
				errors.Token("version", versionID),
			)
		}

		c.secretsCache[cacheKey] = fetchedSecret
		secret = fetchedSecret
	}

	if len(secret.GetEntries()) == 0 {
		return "", errors.Error("secret has no entries")
	}

	for _, entry := range secret.GetEntries() {
		if entry.GetKey() == key {
			return entry.GetTextValue(), nil
		}
	}

	return "", errors.Errorf("key not found in secret %v", errors.Token("key", key))
}

func loadConfig(
	ctx context.Context,
	env Environment,
	ycSDKClient *ycsdk.SDK,
) (Config, error) {
	configsPath := os.Getenv("CONFIGS_PATH")
	fmt.Println("CONFIGS_PATH", configsPath)

	v := viper.New()

	v.AddConfigPath(configsPath)
	v.SetConfigType("yaml")

	var configs []string
	addIfExists := func(name string) {
		fileName := fmt.Sprintf("%s.yaml", name)
		path := filepath.Join(configsPath, fileName)
		if _, err := os.Stat(path); err == nil { //#nosec G703
			configs = append(configs, name)
		}
	}

	addIfExists("base")
	addIfExists(env.String())

	if len(configs) == 0 {
		return Config{}, fmt.Errorf("no config found in %s", configsPath)
	}

	err := createConfig(v, configs)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		viperLoader:  v,
		ycSDKClient:  ycSDKClient,
		secretsCache: make(map[string]*lockbox.Payload),
	}

	err = cfg.readSection(ctx, "grpc", &cfg.grpc)
	if err != nil {
		return Config{}, errors.Wrap(err, "load grpc server config")
	}

	err = cfg.readSection(ctx, "http", &cfg.http)
	if err != nil {
		return Config{}, errors.Wrap(err, "load http server config")
	}

	err = cfg.readSection(ctx, "logging", &cfg.logging)
	if err != nil {
		return Config{}, errors.Wrap(err, "load logging config")
	}

	err = cfg.readSection(ctx, "swagger-ui", &cfg.swaggerUI)
	if err != nil {
		return Config{}, errors.WrapFail(err, "load swagger-ui config")
	}

	return cfg, nil
}

func createConfig(v *viper.Viper, configs []string) error {
	v.SetConfigName(configs[0])
	if err := v.ReadInConfig(); err != nil {
		return errors.WrapFailf(err, "read %v", errors.Token("config", configs[0]))
	}
	for i := 1; i < len(configs); i++ {
		v.SetConfigName(configs[i])
		if err := v.MergeInConfig(); err != nil {
			return errors.WrapFailf(err, "merge %v", errors.Token("config", configs[i]))
		}
	}

	return nil
}
