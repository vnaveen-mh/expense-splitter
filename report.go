package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SaveReportInput struct {
	Markdown string `json:"markdown,omitempty" jsonschema_description:"Report in markdown format"`
}

type SaveReportOutput struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

func SaveReport(ctx context.Context, req *mcp.CallToolRequest, input *SaveReportInput) (result *mcp.CallToolResult, output *SaveReportOutput, err error) {
	md := ""
	if input != nil {
		md = input.Markdown
	}

	start := logToolEnter("SaveReport", "bytes=%d", len(md))
	defer func() {
		logToolExit("SaveReport", start, err, result, output)
	}()

	if strings.TrimSpace(md) == "" {
		return nil, nil, fmt.Errorf("markdown is required")
	}

	id := uniqueReportID()
	baseDir := "reports"
	filename := id + ".md"

	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, nil, err
	}

	path := filepath.Join(baseDir, filename)
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, baseDir+string(filepath.Separator)) {
		return nil, nil, fmt.Errorf("invalid report path")
	}

	if err := os.WriteFile(cleanPath, []byte(md), 0o644); err != nil {
		return nil, nil, err
	}

	output = &SaveReportOutput{
		ID:   id,
		Path: cleanPath,
	}
	return nil, output, nil
}

func uniqueReportID() string {
	ts := time.Now().UTC().Format("20060102T150405.000000000Z")
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ts
	}
	return fmt.Sprintf("%s-%s", ts, hex.EncodeToString(b[:]))
}
