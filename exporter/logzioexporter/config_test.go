// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logzioexporter

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func TestLoadConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	sub, err := cm.Sub(config.NewComponentIDWithName(typeStr, "2").String())
	require.NoError(t, err)
	require.NoError(t, config.UnmarshalExporter(sub, cfg))

	expected := &Config{
		ExporterSettings: config.NewExporterSettings(config.NewComponentID(typeStr)),
		Token:            "token",
		Region:           "eu",
	}
	expected.RetrySettings = exporterhelper.NewDefaultRetrySettings()
	expected.RetrySettings.MaxInterval = 5 * time.Second
	expected.QueueSettings = exporterhelper.NewDefaultQueueSettings()
	expected.QueueSettings.Enabled = false
	expected.HTTPClientSettings = confighttp.HTTPClientSettings{
		Endpoint: "",
		Timeout:  30 * time.Second,
		Headers:  map[string]string{},
		// Default to gzip compression
		Compression: configcompression.Gzip,
		// We almost read 0 bytes, so no need to tune ReadBufferSize.
		WriteBufferSize: 512 * 1024,
	}
	assert.Equal(t, expected, cfg)
}

func TestDefaultLoadConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "configd.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	sub, err := cm.Sub(config.NewComponentIDWithName(typeStr, "2").String())
	require.NoError(t, err)
	require.NoError(t, config.UnmarshalExporter(sub, cfg))

	expected := &Config{
		ExporterSettings: config.NewExporterSettings(config.NewComponentID(typeStr)),
		Token:            "logzioTESTtoken",
	}
	expected.RetrySettings = exporterhelper.NewDefaultRetrySettings()
	expected.QueueSettings = exporterhelper.NewDefaultQueueSettings()
	expected.HTTPClientSettings = confighttp.HTTPClientSettings{
		Endpoint: "",
		Timeout:  30 * time.Second,
		Headers:  map[string]string{},
		// Default to gzip compression
		Compression: configcompression.Gzip,
		// We almost read 0 bytes, so no need to tune ReadBufferSize.
		WriteBufferSize: 512 * 1024,
	}
	assert.Equal(t, expected, cfg)
}

func TestCheckAndWarnDeprecatedOptions(t *testing.T) {
	// Config with legacy options
	actualCfg := &Config{
		ExporterSettings: config.NewExporterSettings(config.NewComponentIDWithName(typeStr, "2")),
		QueueSettings:    exporterhelper.NewDefaultQueueSettings(),
		RetrySettings:    exporterhelper.NewDefaultRetrySettings(),
		Token:            "logzioTESTtoken",
		CustomEndpoint:   "https://api.example.com",
		QueueMaxLength:   10,
		DrainInterval:    10,
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: "",
			Timeout:  10 * time.Second,
			Headers:  map[string]string{},
			// Default to gzip compression
			Compression: configcompression.Gzip,
			// We almost read 0 bytes, so no need to tune ReadBufferSize.
			WriteBufferSize: 512 * 1024,
		},
	}
	params := componenttest.NewNopExporterCreateSettings()
	logger := hclog2ZapLogger{
		Zap:  params.Logger,
		name: loggerName,
	}
	actualCfg.checkAndWarnDeprecatedOptions(&logger)

	expected := &Config{
		ExporterSettings: config.NewExporterSettings(config.NewComponentIDWithName(typeStr, "2")),
		Token:            "logzioTESTtoken",
		CustomEndpoint:   "https://api.example.com",
		QueueMaxLength:   10,
		DrainInterval:    10,
		RetrySettings:    exporterhelper.NewDefaultRetrySettings(),
		QueueSettings:    exporterhelper.NewDefaultQueueSettings(),
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: "https://api.example.com",
			Timeout:  10 * time.Second,
			Headers:  map[string]string{},
			// Default to gzip compression
			Compression: configcompression.Gzip,
			// We almost read 0 bytes, so no need to tune ReadBufferSize.
			WriteBufferSize: 512 * 1024,
		},
	}
	expected.QueueSettings.QueueSize = 10
	assert.Equal(t, expected, actualCfg)
}
