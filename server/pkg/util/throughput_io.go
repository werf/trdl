package util

import (
	"context"
	"io"
	"time"

	"github.com/werf/logboek"
)

type ThroughputWriter struct {
	LogName        string
	W              io.Writer
	StartedAt      time.Time
	LastProgressAt time.Time
	ProcessedBytes int
	Context        context.Context
}

func NewThroughputWriter(logName string, w io.Writer, ctx context.Context) *ThroughputWriter {
	return &ThroughputWriter{LogName: logName, W: w, Context: ctx}
}

func (w *ThroughputWriter) Write(p []byte) (int, error) {
	now := time.Now()

	if w.StartedAt.IsZero() {
		w.StartedAt = now
	}

	w.ProcessedBytes += len(p)
	durSecs := int(now.Sub(w.StartedAt) / time.Second)

	if now.Sub(w.LastProgressAt) > time.Second && durSecs > 0 {
		logboek.Context(w.Context).Default().LogF("[%s] Writer processed durSecs=%d %d bytes, speed %d Mb/min\n", w.LogName, durSecs, w.ProcessedBytes, int(60*(float64(w.ProcessedBytes)/float64(durSecs))/1024.0/1024.0))
		w.LastProgressAt = now
	}

	return w.W.Write(p)
}

type ThroughputReader struct {
	LogName        string
	R              io.Reader
	StartedAt      time.Time
	LastProgressAt time.Time
	ProcessedBytes int
	Context        context.Context
}

func NewThroughputReader(logName string, r io.Reader, ctx context.Context) *ThroughputReader {
	return &ThroughputReader{LogName: logName, R: r, Context: ctx}
}

func (r *ThroughputReader) Read(p []byte) (int, error) {
	n, err := r.R.Read(p)

	if err == nil || err == io.EOF {
		now := time.Now()

		if r.StartedAt.IsZero() {
			r.StartedAt = now
		}

		r.ProcessedBytes += n
		durSecs := int(now.Sub(r.StartedAt) / time.Second)

		if now.Sub(r.LastProgressAt) > time.Second && durSecs > 0 {
			logboek.Context(r.Context).Default().LogF("[%s] Reader processed durSecs=%d %d bytes, speed %d Mb/min\n", r.LogName, durSecs, r.ProcessedBytes, int(60*(float64(r.ProcessedBytes)/float64(durSecs))/1024.0/1024.0))
			r.LastProgressAt = now
		}
	}

	return n, err
}
