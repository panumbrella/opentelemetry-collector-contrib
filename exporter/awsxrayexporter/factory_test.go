// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package awsxrayexporter

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/confmap/confmaptest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/awsutil"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, cfg, &Config{
		ExporterSettings: config.NewExporterSettings(config.NewComponentID(typeStr)),
		AWSSessionSettings: awsutil.AWSSessionSettings{
			NumberOfWorkers:       8,
			Endpoint:              "",
			RequestTimeoutSeconds: 30,
			MaxRetries:            2,
			NoVerifySSL:           false,
			ProxyAddress:          "",
			Region:                "",
			LocalMode:             false,
			ResourceARN:           "",
			RoleARN:               "",
		},
	}, "failed to create default config")
	assert.NoError(t, configtest.CheckConfigStruct(cfg))
}

func TestCreateTracesExporter(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	sub, err := cm.Sub(config.NewComponentIDWithName(typeStr, "customname").String())
	require.NoError(t, err)
	require.NoError(t, config.UnmarshalExporter(sub, cfg))

	ctx := context.Background()
	exporter, err := factory.CreateTracesExporter(ctx, componenttest.NewNopExporterCreateSettings(), cfg)
	assert.Nil(t, err)
	assert.NotNil(t, exporter)
}

func TestCreateMetricsExporter(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	sub, err := cm.Sub(config.NewComponentIDWithName(typeStr, "customname").String())
	require.NoError(t, err)
	require.NoError(t, config.UnmarshalExporter(sub, cfg))

	ctx := context.Background()
	exporter, err := factory.CreateMetricsExporter(ctx, componenttest.NewNopExporterCreateSettings(), cfg)
	assert.NotNil(t, err)
	assert.Nil(t, exporter)
}
