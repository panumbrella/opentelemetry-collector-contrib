// Copyright 2020, OpenTelemetry Authors
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

package jmxreceiver

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	initSupportedJars()
	tests := []struct {
		id          config.ComponentID
		expected    config.Receiver
		expectedErr string
	}{
		{
			id:          config.NewComponentIDWithName(typeStr, ""),
			expectedErr: "missing required fields: `endpoint`, `target_system`",
			expected:    createDefaultConfig(),
		},
		{
			id: config.NewComponentIDWithName(typeStr, "all"),
			expected: &Config{
				ReceiverSettings:   config.NewReceiverSettings(config.NewComponentID(typeStr)),
				JARPath:            "testdata/fake_jmx.jar",
				Endpoint:           "myendpoint:12345",
				TargetSystem:       "jvm",
				CollectionInterval: 15 * time.Second,
				Username:           "myusername",
				Password:           "mypassword",
				LogLevel:           "trace",
				OTLPExporterConfig: otlpExporterConfig{
					Endpoint: "myotlpendpoint",
					Headers: map[string]string{
						"x-header-1": "value1",
						"x-header-2": "value2",
					},
					TimeoutSettings: exporterhelper.TimeoutSettings{
						Timeout: 5 * time.Second,
					},
				},
				KeystorePath:       "mykeystorepath",
				KeystorePassword:   "mykeystorepassword",
				KeystoreType:       "mykeystoretype",
				TruststorePath:     "mytruststorepath",
				TruststorePassword: "mytruststorepassword",
				RemoteProfile:      "myremoteprofile",
				Realm:              "myrealm",
				AdditionalJars: []string{
					"testdata/fake_additional.jar",
				},
				ResourceAttributes: map[string]string{
					"one": "two",
				},
			},
		},
		{
			id: config.NewComponentIDWithName(typeStr, "missingendpoint"),
			expected: &Config{
				ReceiverSettings:   config.NewReceiverSettings(config.NewComponentID(typeStr)),
				JARPath:            "testdata/fake_jmx.jar",
				TargetSystem:       "jvm",
				CollectionInterval: 10 * time.Second,
				OTLPExporterConfig: otlpExporterConfig{
					Endpoint: "0.0.0.0:0",
					TimeoutSettings: exporterhelper.TimeoutSettings{
						Timeout: 5 * time.Second,
					},
				},
			},
		},
		{
			id:          config.NewComponentIDWithName(typeStr, "missingtarget"),
			expectedErr: "jmx missing required field: `target_system`",
			expected: &Config{
				ReceiverSettings:   config.NewReceiverSettings(config.NewComponentID(typeStr)),
				JARPath:            "testdata/fake_jmx.jar",
				Endpoint:           "service:jmx:rmi:///jndi/rmi://host:12345/jmxrmi",
				CollectionInterval: 10 * time.Second,
				OTLPExporterConfig: otlpExporterConfig{
					Endpoint: "0.0.0.0:0",
					TimeoutSettings: exporterhelper.TimeoutSettings{
						Timeout: 5 * time.Second,
					},
				},
			},
		},
		{
			id:          config.NewComponentIDWithName(typeStr, "invalidinterval"),
			expectedErr: "`interval` must be positive: -100ms",
			expected: &Config{
				ReceiverSettings:   config.NewReceiverSettings(config.NewComponentID(typeStr)),
				JARPath:            "testdata/fake_jmx.jar",
				Endpoint:           "myendpoint:23456",
				TargetSystem:       "jvm",
				CollectionInterval: -100 * time.Millisecond,
				OTLPExporterConfig: otlpExporterConfig{
					Endpoint: "0.0.0.0:0",
					TimeoutSettings: exporterhelper.TimeoutSettings{
						Timeout: 5 * time.Second,
					},
				},
			},
		},
		{
			id:          config.NewComponentIDWithName(typeStr, "invalidotlptimeout"),
			expectedErr: "`otlp.timeout` must be positive: -100ms",
			expected: &Config{
				ReceiverSettings:   config.NewReceiverSettings(config.NewComponentID(typeStr)),
				JARPath:            "testdata/fake_jmx.jar",
				Endpoint:           "myendpoint:34567",
				TargetSystem:       "jvm",
				CollectionInterval: 10 * time.Second,
				OTLPExporterConfig: otlpExporterConfig{
					Endpoint: "0.0.0.0:0",
					TimeoutSettings: exporterhelper.TimeoutSettings{
						Timeout: -100 * time.Millisecond,
					},
				},
			},
		},

		{
			id: config.NewComponentIDWithName(typeStr, "nonexistentjar"),
			// Error is different based on OS, which is why this is contains, not equals
			expectedErr: "error validating `jar_path`: error hashing file: open testdata/file_does_not_exist.jar:",
			expected: &Config{
				ReceiverSettings:   config.NewReceiverSettings(config.NewComponentID(typeStr)),
				JARPath:            "testdata/file_does_not_exist.jar",
				Endpoint:           "myendpoint:23456",
				TargetSystem:       "jvm",
				CollectionInterval: 10 * time.Second,
				OTLPExporterConfig: otlpExporterConfig{
					Endpoint: "0.0.0.0:0",
					TimeoutSettings: exporterhelper.TimeoutSettings{
						Timeout: 5 * time.Second,
					},
				},
			},
		},
		{
			id:          config.NewComponentIDWithName(typeStr, "invalidjar"),
			expectedErr: "error validating `jar_path`: jar hash does not match known versions",
			expected: &Config{
				ReceiverSettings:   config.NewReceiverSettings(config.NewComponentID(typeStr)),
				JARPath:            "testdata/fake_jmx_wrong.jar",
				Endpoint:           "myendpoint:23456",
				TargetSystem:       "jvm",
				CollectionInterval: 10 * time.Second,
				OTLPExporterConfig: otlpExporterConfig{
					Endpoint: "0.0.0.0:0",
					TimeoutSettings: exporterhelper.TimeoutSettings{
						Timeout: 5 * time.Second,
					},
				},
			},
		},
		{
			id:          config.NewComponentIDWithName(typeStr, "invalidloglevel"),
			expectedErr: "jmx `log_level` must be one of 'debug', 'error', 'info', 'off', 'trace', 'warn'",
			expected: &Config{
				ReceiverSettings:   config.NewReceiverSettings(config.NewComponentID(typeStr)),
				JARPath:            "testdata/fake_jmx.jar",
				Endpoint:           "myendpoint:55555",
				TargetSystem:       "jvm",
				LogLevel:           "truth",
				CollectionInterval: 10 * time.Second,
				OTLPExporterConfig: otlpExporterConfig{
					Endpoint: "0.0.0.0:0",
					TimeoutSettings: exporterhelper.TimeoutSettings{
						Timeout: 5 * time.Second,
					},
				},
			},
		},
		{
			id:          config.NewComponentIDWithName(typeStr, "invalidtargetsystem"),
			expectedErr: "`target_system` list may only be a subset of 'activemq', 'cassandra', 'hadoop', 'hbase', 'jetty', 'jvm', 'kafka', 'kafka-consumer', 'kafka-producer', 'solr', 'tomcat', 'wildfly'",
			expected: &Config{
				ReceiverSettings:   config.NewReceiverSettings(config.NewComponentID(typeStr)),
				JARPath:            "testdata/fake_jmx.jar",
				Endpoint:           "myendpoint:55555",
				TargetSystem:       "jvm,fakejvmtechnology",
				CollectionInterval: 10 * time.Second,
				OTLPExporterConfig: otlpExporterConfig{
					Endpoint: "0.0.0.0:0",
					TimeoutSettings: exporterhelper.TimeoutSettings{
						Timeout: 5 * time.Second,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.id.String(), func(t *testing.T) {
			mockJarVersions()
			t.Cleanup(func() {
				unmockJarVersions()
			})

			factory := NewFactory()
			cfg := factory.CreateDefaultConfig()

			sub, err := cm.Sub(tt.id.String())
			require.NoError(t, err)
			require.NoError(t, config.UnmarshalReceiver(sub, cfg))

			if tt.expectedErr != "" {
				assert.ErrorContains(t, cfg.(*Config).validate(), tt.expectedErr)
				assert.Equal(t, tt.expected, cfg)
				return
			}
			assert.NoError(t, cfg.Validate())
			assert.Equal(t, tt.expected, cfg)
		})
	}
}

func TestCustomMetricsGathererConfig(t *testing.T) {
	wildflyJarVersions["7d1a54127b222502f5b79b5fb0803061152a44f92b37e23c6527baf665d4da9a"] = supportedJar{
		jar:     "fake wildfly jar",
		version: "2.3.4",
	}

	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	sub, err := cm.Sub(config.NewComponentIDWithName(typeStr, "invalidtargetsystem").String())
	require.NoError(t, err)
	require.NoError(t, config.UnmarshalReceiver(sub, cfg))

	conf := cfg.(*Config)

	err = conf.validate()
	require.Error(t, err)
	assert.Equal(t, "jmx error validating `jar_path`: jar hash does not match known versions", err.Error())

	MetricsGathererHash = "5994471abb01112afcc18159f6cc74b4f511b99806da59b3caf5a9c173cacfc5"
	initSupportedJars()

	err = conf.validate()
	require.Error(t, err)
	assert.Equal(t, "jmx `target_system` list may only be a subset of 'activemq', 'cassandra', 'hadoop', 'hbase', 'jetty', 'jvm', 'kafka', 'kafka-consumer', 'kafka-producer', 'solr', 'tomcat', 'wildfly'", err.Error())

	AdditionalTargetSystems = "fakejvmtechnology,anothertechnology"
	t.Cleanup(func() {
		delete(validTargetSystems, "fakejvmtechnology")
		delete(validTargetSystems, "anothertechnology")
	})
	initAdditionalTargetSystems()

	conf.TargetSystem = "jvm,fakejvmtechnology,anothertechnology"

	require.NoError(t, conf.validate())
}

func TestClassPathParse(t *testing.T) {
	testCases := []struct {
		desc           string
		cfg            *Config
		existingEnvVal string
		expected       string
	}{
		{
			desc: "Metric Gatherer JAR Only",
			cfg: &Config{
				JARPath: "testdata/fake_jmx.jar",
			},
			existingEnvVal: "",
			expected:       "testdata/fake_jmx.jar",
		},
		{
			desc: "Additional JARS",
			cfg: &Config{
				JARPath: "testdata/fake_jmx.jar",
				AdditionalJars: []string{
					"/path/to/one.jar",
					"/path/to/two.jar",
				},
			},
			existingEnvVal: "",
			expected:       "testdata/fake_jmx.jar:/path/to/one.jar:/path/to/two.jar",
		},
		{
			desc: "Existing ENV Value",
			cfg: &Config{
				JARPath: "testdata/fake_jmx.jar",
				AdditionalJars: []string{
					"/path/to/one.jar",
					"/path/to/two.jar",
				},
			},
			existingEnvVal: "/pre/existing/class/path/",
			expected:       "testdata/fake_jmx.jar:/path/to/one.jar:/path/to/two.jar",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Setenv("CLASSPATH", tc.existingEnvVal)

			actual := tc.cfg.parseClasspath()
			require.Equal(t, tc.expected, actual)
		})
	}
}

func mockJarVersions() {
	jmxMetricsGathererVersions["5994471abb01112afcc18159f6cc74b4f511b99806da59b3caf5a9c173cacfc5"] = supportedJar{
		jar:     "fake jar",
		version: "1.2.3",
	}

	wildflyJarVersions["7d1a54127b222502f5b79b5fb0803061152a44f92b37e23c6527baf665d4da9a"] = supportedJar{
		jar:     "fake wildfly jar",
		version: "2.3.4",
	}
}

func unmockJarVersions() {
	delete(jmxMetricsGathererVersions, "5994471abb01112afcc18159f6cc74b4f511b99806da59b3caf5a9c173cacfc5")
	delete(wildflyJarVersions, "7d1a54127b222502f5b79b5fb0803061152a44f92b37e23c6527baf665d4da9a")
}
