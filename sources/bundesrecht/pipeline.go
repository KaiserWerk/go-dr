package bundesrecht

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	godr "github.com/KaiserWerk/go-dr"
)

const (
	defaultWorkerCount = 4
	defaultMaxFailures = 100
)

// DocumentSink persists or forwards fetched documents in bulk pipelines.
type DocumentSink interface {
	SaveDocument(ctx context.Context, sourceName string, ref godr.DocumentRef, doc *godr.LegalDocument) error
}

// SinkFunc adapts a function to DocumentSink.
type SinkFunc func(ctx context.Context, sourceName string, ref godr.DocumentRef, doc *godr.LegalDocument) error

// SaveDocument implements DocumentSink.
func (f SinkFunc) SaveDocument(ctx context.Context, sourceName string, ref godr.DocumentRef, doc *godr.LegalDocument) error {
	if f == nil {
		return nil
	}
	return f(ctx, sourceName, ref, doc)
}

// ProgressStore tracks completion state for resumable ingestion.
type ProgressStore interface {
	IsCompleted(ctx context.Context, sourceName, documentKey string) (bool, error)
	MarkCompleted(ctx context.Context, sourceName, documentKey string) error
	MarkFailed(ctx context.Context, sourceName, documentKey, reason string) error
}

// PipelineConfig configures end-to-end Bundesrecht completeness ingestion.
type PipelineConfig struct {
	Source          Source
	Sink            DocumentSink
	Progress        ProgressStore
	Workers         int
	ContinueOnError bool
	MaxFailures     int
}

// PipelineFailure describes one failed fetch/store operation.
type PipelineFailure struct {
	DocumentKey string
	URL         string
	Reason      string
}

// CompletenessReport summarizes one full Bundesrecht pipeline run.
type CompletenessReport struct {
	SourceName        string
	StartedAt         time.Time
	FinishedAt        time.Time
	Duration          time.Duration
	TotalListed       int
	TotalQueued       int
	TotalFetched      int
	TotalStored       int
	TotalFailed       int
	SkippedCompleted  int
	Failed            []PipelineFailure
	MissingDocumentID []string
}

type runState struct {
	mu           sync.Mutex
	failed       []PipelineFailure
	missingIDs   map[string]struct{}
	totalQueued  int
	totalFetched int
	totalStored  int
	totalFailed  int
	skippedDone  int
	firstErr     error
}

// Coverage returns fetched/listed coverage in [0,1].
func (r CompletenessReport) Coverage() float64 {
	if r.TotalListed == 0 {
		return 1
	}
	return float64(r.TotalFetched) / float64(r.TotalListed)
}

// IsComplete reports whether all listed documents were fetched successfully.
func (r CompletenessReport) IsComplete() bool {
	return r.TotalListed > 0 && r.TotalFailed == 0 && r.TotalFetched == r.TotalListed
}

// RunPipeline executes complete Bundesrecht listing/fetching with optional persistence and resume support.
func RunPipeline(ctx context.Context, cfg PipelineConfig) (CompletenessReport, error) {
	src := cfg.Source
	sourceName := strings.TrimSpace(src.Name())
	if sourceName == "" {
		sourceName = "bundesrecht"
	}

	workers := cfg.Workers
	if workers <= 0 {
		workers = defaultWorkerCount
	}
	maxFailures := cfg.MaxFailures
	if maxFailures <= 0 {
		maxFailures = defaultMaxFailures
	}

	report := CompletenessReport{
		SourceName: sourceName,
		StartedAt:  time.Now().UTC(),
		Failed:     make([]PipelineFailure, 0),
	}

	refs, err := src.ListDocuments(ctx)
	if err != nil {
		report.FinishedAt = time.Now().UTC()
		report.Duration = report.FinishedAt.Sub(report.StartedAt)
		return report, err
	}

	report.TotalListed = len(refs)
	refs = uniqueRefs(refs)

	jobs := make(chan godr.DocumentRef)
	ctxRun := ctx
	var cancel context.CancelFunc
	if !cfg.ContinueOnError {
		ctxRun, cancel = context.WithCancel(ctx)
		defer cancel()
	}

	state := &runState{missingIDs: make(map[string]struct{}, len(refs))}
	for _, ref := range refs {
		state.missingIDs[documentKey(ref)] = struct{}{}
	}

	workerFn := func() {
		for ref := range jobs {
			if ctxRun.Err() != nil {
				return
			}

			key := documentKey(ref)
			if cfg.Progress != nil {
				done, err := cfg.Progress.IsCompleted(ctxRun, sourceName, key)
				if err != nil {
					state.recordFailure(maxFailures, PipelineFailure{DocumentKey: key, URL: ref.URL, Reason: err.Error()})
					state.setFirstErr(fmt.Errorf("progress check failed for %s: %w", key, err), cancel)
					if !cfg.ContinueOnError {
						return
					}
					continue
				}
				if done {
					state.markSkipped(key)
					continue
				}
			}

			doc, err := src.FetchDocument(ctxRun, ref)
			if err != nil {
				reason := err.Error()
				state.recordFailure(maxFailures, PipelineFailure{DocumentKey: key, URL: ref.URL, Reason: reason})
				if cfg.Progress != nil {
					_ = cfg.Progress.MarkFailed(ctxRun, sourceName, key, reason)
				}
				state.setFirstErr(fmt.Errorf("fetch failed for %s: %w", key, err), cancel)
				if !cfg.ContinueOnError {
					return
				}
				continue
			}

			state.markFetched(key)

			if cfg.Sink != nil {
				if err := cfg.Sink.SaveDocument(ctxRun, sourceName, ref, doc); err != nil {
					reason := err.Error()
					state.recordFailure(maxFailures, PipelineFailure{DocumentKey: key, URL: ref.URL, Reason: reason})
					if cfg.Progress != nil {
						_ = cfg.Progress.MarkFailed(ctxRun, sourceName, key, reason)
					}
					state.setFirstErr(fmt.Errorf("sink failed for %s: %w", key, err), cancel)
					if !cfg.ContinueOnError {
						return
					}
					continue
				}
				state.markStored()
			}

			if cfg.Progress != nil {
				if err := cfg.Progress.MarkCompleted(ctxRun, sourceName, key); err != nil {
					state.recordFailure(maxFailures, PipelineFailure{DocumentKey: key, URL: ref.URL, Reason: err.Error()})
					state.setFirstErr(fmt.Errorf("mark completed failed for %s: %w", key, err), cancel)
					if !cfg.ContinueOnError {
						return
					}
				}
			}
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			workerFn()
		}()
	}

	for _, ref := range refs {
		if ctxRun.Err() != nil {
			break
		}
		state.markQueued()
		jobs <- ref
	}
	close(jobs)
	wg.Wait()

	report.FinishedAt = time.Now().UTC()
	report.Duration = report.FinishedAt.Sub(report.StartedAt)
	report.TotalQueued = state.totalQueued
	report.TotalFetched = state.totalFetched
	report.TotalStored = state.totalStored
	report.TotalFailed = state.totalFailed
	report.SkippedCompleted = state.skippedDone
	report.Failed = append(report.Failed, state.failed...)
	report.MissingDocumentID = state.missingList()

	if cfg.ContinueOnError {
		return report, nil
	}
	if state.firstErr != nil {
		return report, state.firstErr
	}
	if ctxRun.Err() != nil {
		return report, ctxRun.Err()
	}

	return report, nil
}

func (s *runState) markQueued() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalQueued++
}

func (s *runState) markSkipped(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.skippedDone++
	delete(s.missingIDs, key)
}

func (s *runState) markFetched(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalFetched++
	delete(s.missingIDs, key)
}

func (s *runState) markStored() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalStored++
}

func (s *runState) recordFailure(maxFailures int, failure PipelineFailure) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalFailed++
	if len(s.failed) < maxFailures {
		s.failed = append(s.failed, failure)
	}
}

func (s *runState) setFirstErr(err error, cancel context.CancelFunc) {
	if err == nil {
		return
	}
	s.mu.Lock()
	if s.firstErr == nil {
		s.firstErr = err
		if cancel != nil {
			cancel()
		}
	}
	s.mu.Unlock()
}

func (s *runState) missingList() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.missingIDs))
	for k := range s.missingIDs {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func uniqueRefs(in []godr.DocumentRef) []godr.DocumentRef {
	seen := make(map[string]struct{}, len(in))
	out := make([]godr.DocumentRef, 0, len(in))
	for _, ref := range in {
		key := documentKey(ref)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func documentKey(ref godr.DocumentRef) string {
	if id := strings.TrimSpace(ref.ID); id != "" {
		return id
	}
	return strings.TrimSpace(ref.URL)
}

type progressSnapshot map[string]progressRecord

type progressRecord struct {
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
	UpdatedAt string `json:"updated_at"`
}

// FileProgressStore persists resumable ingest progress in one JSON file.
type FileProgressStore struct {
	Path string
	mu   sync.Mutex
}

// IsCompleted reports whether a document key is already marked as completed.
func (s *FileProgressStore) IsCompleted(_ context.Context, sourceName, documentKey string) (bool, error) {
	recs, err := s.load()
	if err != nil {
		return false, err
	}
	entry, ok := recs[progressKey(sourceName, documentKey)]
	return ok && entry.Status == "completed", nil
}

// MarkCompleted writes a completed progress marker.
func (s *FileProgressStore) MarkCompleted(_ context.Context, sourceName, documentKey string) error {
	return s.update(sourceName, documentKey, progressRecord{Status: "completed", UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
}

// MarkFailed writes a failed progress marker with reason.
func (s *FileProgressStore) MarkFailed(_ context.Context, sourceName, documentKey, reason string) error {
	return s.update(sourceName, documentKey, progressRecord{Status: "failed", Reason: strings.TrimSpace(reason), UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
}

func (s *FileProgressStore) update(sourceName, documentKey string, record progressRecord) error {
	if strings.TrimSpace(sourceName) == "" {
		return errors.New("source name is required")
	}
	if strings.TrimSpace(documentKey) == "" {
		return errors.New("document key is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	recs, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	recs[progressKey(sourceName, documentKey)] = record
	return s.saveUnlocked(recs)
}

func (s *FileProgressStore) load() (progressSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadUnlocked()
}

func (s *FileProgressStore) loadUnlocked() (progressSnapshot, error) {
	path := strings.TrimSpace(s.Path)
	if path == "" {
		return nil, errors.New("file progress store requires path")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(progressSnapshot), nil
		}
		return nil, err
	}
	if len(raw) == 0 {
		return make(progressSnapshot), nil
	}

	out := make(progressSnapshot)
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = make(progressSnapshot)
	}
	return out, nil
}

func (s *FileProgressStore) save(records progressSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUnlocked(records)
}

func (s *FileProgressStore) saveUnlocked(records progressSnapshot) error {
	path := strings.TrimSpace(s.Path)
	if path == "" {
		return errors.New("file progress store requires path")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func progressKey(sourceName, documentKey string) string {
	return strings.TrimSpace(sourceName) + "|" + strings.TrimSpace(documentKey)
}
