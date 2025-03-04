// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package processscraper // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/processscraper"

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver/scrapererror"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/processor/filterset"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/scraper/processscraper/internal/metadata"
)

const (
	cpuMetricsLen    = 1
	memoryMetricsLen = 2
	diskMetricsLen   = 1
	pagingMetricsLen = 1
	threadMetricsLen = 1

	metricsLen = cpuMetricsLen + memoryMetricsLen + diskMetricsLen + pagingMetricsLen + threadMetricsLen
)

// scraper for Process Metrics
type scraper struct {
	settings           component.ReceiverCreateSettings
	config             *Config
	mb                 *metadata.MetricsBuilder
	includeFS          filterset.FilterSet
	excludeFS          filterset.FilterSet
	scrapeProcessDelay time.Duration
	// for mocking
	getProcessCreateTime func(p processHandle) (int64, error)
	getProcessHandles    func() (processHandles, error)
}

// newProcessScraper creates a Process Scraper
func newProcessScraper(settings component.ReceiverCreateSettings, cfg *Config) (*scraper, error) {
	scraper := &scraper{
		settings:             settings,
		config:               cfg,
		getProcessCreateTime: processHandle.CreateTime,
		getProcessHandles:    getProcessHandlesInternal,
		scrapeProcessDelay:   cfg.ScrapeProcessDelay,
	}

	var err error

	if len(cfg.Include.Names) > 0 {
		scraper.includeFS, err = filterset.CreateFilterSet(cfg.Include.Names, &cfg.Include.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating process include filters: %w", err)
		}
	}

	if len(cfg.Exclude.Names) > 0 {
		scraper.excludeFS, err = filterset.CreateFilterSet(cfg.Exclude.Names, &cfg.Exclude.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating process exclude filters: %w", err)
		}
	}

	return scraper, nil
}

func (s *scraper) start(context.Context, component.Host) error {
	s.mb = metadata.NewMetricsBuilder(s.config.Metrics, s.settings.BuildInfo)
	return nil
}

func (s *scraper) scrape(_ context.Context) (pmetric.Metrics, error) {
	var errs scrapererror.ScrapeErrors

	data, err := s.getProcessMetadata()
	if err != nil {
		var partialErr scrapererror.PartialScrapeError
		if !errors.As(err, &partialErr) {
			return pmetric.NewMetrics(), err
		}

		errs.AddPartial(partialErr.Failed, partialErr)
	}

	for _, md := range data {
		now := pcommon.NewTimestampFromTime(time.Now())

		if err = s.scrapeAndAppendCPUTimeMetric(now, md.handle); err != nil {
			errs.AddPartial(cpuMetricsLen, fmt.Errorf("error reading cpu times for process %q (pid %v): %w", md.executable.name, md.pid, err))
		}

		if err = s.scrapeAndAppendMemoryUsageMetrics(now, md.handle); err != nil {
			errs.AddPartial(memoryMetricsLen, fmt.Errorf("error reading memory info for process %q (pid %v): %w", md.executable.name, md.pid, err))
		}

		if err = s.scrapeAndAppendDiskIOMetric(now, md.handle); err != nil {
			errs.AddPartial(diskMetricsLen, fmt.Errorf("error reading disk usage for process %q (pid %v): %w", md.executable.name, md.pid, err))
		}

		if err = s.scrapeAndAppendPagingMetric(now, md.handle); err != nil {
			errs.AddPartial(pagingMetricsLen, fmt.Errorf("error reading memory paging info for process %q (pid %v): %w", md.executable.name, md.pid, err))
		}

		if err = s.scrapeAndAppendThreadsMetrics(now, md.handle); err != nil {
			errs.AddPartial(threadMetricsLen, fmt.Errorf("error reading thread info for process %q (pid %v): %w", md.executable.name, md.pid, err))
		}

		options := append(md.resourceOptions(), metadata.WithStartTimeOverride(pcommon.Timestamp(md.createTime*1e6)))
		s.mb.EmitForResource(options...)
	}

	return s.mb.Emit(), errs.Combine()
}

// getProcessMetadata returns a slice of processMetadata, including handles,
// for all currently running processes. If errors occur obtaining information
// for some processes, an error will be returned, but any processes that were
// successfully obtained will still be returned.
func (s *scraper) getProcessMetadata() ([]*processMetadata, error) {
	handles, err := s.getProcessHandles()
	if err != nil {
		return nil, err
	}

	var errs scrapererror.ScrapeErrors

	data := make([]*processMetadata, 0, handles.Len())
	for i := 0; i < handles.Len(); i++ {
		pid := handles.Pid(i)
		handle := handles.At(i)

		executable, err := getProcessExecutable(handle)
		if err != nil {
			if !s.config.MuteProcessNameError {
				errs.AddPartial(1, fmt.Errorf("error reading process name for pid %v: %w", pid, err))
			}
			continue
		}

		// filter processes by name
		if (s.includeFS != nil && !s.includeFS.Matches(executable.name)) ||
			(s.excludeFS != nil && s.excludeFS.Matches(executable.name)) {
			continue
		}

		command, err := getProcessCommand(handle)
		if err != nil {
			errs.AddPartial(0, fmt.Errorf("error reading command for process %q (pid %v): %w", executable.name, pid, err))
		}

		username, err := handle.Username()
		if err != nil {
			errs.AddPartial(0, fmt.Errorf("error reading username for process %q (pid %v): %w", executable.name, pid, err))
		}

		createTime, err := s.getProcessCreateTime(handle)
		if err != nil {
			errs.AddPartial(0, fmt.Errorf("error reading create time for process %q (pid %v): %w", executable.name, pid, err))
			// set the start time to now to avoid including this when a scrape_process_delay is set
			createTime = time.Now().UnixMilli()
		}
		if s.scrapeProcessDelay.Milliseconds() > (time.Now().UnixMilli() - createTime) {
			continue
		}

		parentPid, err := parentPid(handle, pid)
		if err != nil {
			errs.AddPartial(0, fmt.Errorf("error reading parent pid for process %q (pid %v): %w", executable.name, pid, err))
		}

		md := &processMetadata{
			pid:        pid,
			parentPid:  parentPid,
			executable: executable,
			command:    command,
			username:   username,
			handle:     handle,
			createTime: createTime,
		}

		data = append(data, md)
	}

	return data, errs.Combine()
}

func (s *scraper) scrapeAndAppendCPUTimeMetric(now pcommon.Timestamp, handle processHandle) error {
	times, err := handle.Times()
	if err != nil {
		return err
	}

	s.recordCPUTimeMetric(now, times)
	return nil
}

func (s *scraper) scrapeAndAppendMemoryUsageMetrics(now pcommon.Timestamp, handle processHandle) error {
	mem, err := handle.MemoryInfo()
	if err != nil {
		return err
	}

	s.mb.RecordProcessMemoryPhysicalUsageDataPoint(now, int64(mem.RSS))
	s.mb.RecordProcessMemoryVirtualUsageDataPoint(now, int64(mem.VMS))
	return nil
}

func (s *scraper) scrapeAndAppendDiskIOMetric(now pcommon.Timestamp, handle processHandle) error {
	io, err := handle.IOCounters()
	if err != nil {
		return err
	}

	s.mb.RecordProcessDiskIoDataPoint(now, int64(io.ReadBytes), metadata.AttributeDirectionRead)
	s.mb.RecordProcessDiskIoDataPoint(now, int64(io.WriteBytes), metadata.AttributeDirectionWrite)

	return nil
}

func (s *scraper) scrapeAndAppendPagingMetric(now pcommon.Timestamp, handle processHandle) error {
	if !s.config.Metrics.ProcessPagingFaults.Enabled {
		return nil
	}

	pageFaultsStat, err := handle.PageFaults()
	if err != nil {
		return err
	}

	s.mb.RecordProcessPagingFaultsDataPoint(now, int64(pageFaultsStat.MajorFaults), metadata.AttributeTypeMajor)
	s.mb.RecordProcessPagingFaultsDataPoint(now, int64(pageFaultsStat.MinorFaults), metadata.AttributeTypeMinor)
	return nil
}

func (s *scraper) scrapeAndAppendThreadsMetrics(now pcommon.Timestamp, handle processHandle) error {
	if !s.config.Metrics.ProcessThreads.Enabled {
		return nil
	}
	threads, err := handle.NumThreads()
	if err != nil {
		return err
	}
	s.mb.RecordProcessThreadsDataPoint(now, int64(threads))

	return nil
}
