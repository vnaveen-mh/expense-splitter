package main

import (
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func ptrString(v *string) string {
	if v == nil {
		return "<nil>"
	}
	return *v
}

func logToolEnter(name string, detail string, args ...any) time.Time {
	if detail == "" {
		log.Printf("Tool %s enter", name)
		return time.Now()
	}
	log.Printf("Tool %s enter: "+detail, append([]any{name}, args...)...)
	return time.Now()
}

func logToolExit(name string, start time.Time, err error, result *mcp.CallToolResult, output any) {
	status := "ok"
	if err != nil {
		status = "error"
	} else if result != nil && output == nil {
		status = "cancelled"
	}
	log.Printf("Tool %s exit: status=%s duration=%s", name, status, time.Since(start))
}

func logToolExitWithDetail(name string, start time.Time, err error, result *mcp.CallToolResult, output any, detail string, args ...any) {
	if detail == "" {
		logToolExit(name, start, err, result, output)
		return
	}
	status := "ok"
	if err != nil {
		status = "error"
	} else if result != nil && output == nil {
		status = "cancelled"
	}
	allArgs := append([]any{name, status, time.Since(start)}, args...)
	log.Printf("Tool %s exit: status=%s duration=%s "+detail, allArgs...)
}
