package main

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	version   = "2.0.0"
	commit    = "none"
	buildTime = "unknown"
)

func (a *App) HTTPServer() *http.Server {
	if !a.opts.Verbose {
		gin.SetMode(gin.ReleaseMode)
	}
	return &http.Server{
		Addr:              a.opts.HTTPAddr,
		Handler:           a.router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

func (a *App) router() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery(), a.securityHeaders(), a.corsMiddleware())
	if a.opts.Verbose {
		router.Use(gin.Logger())
	}

	router.GET("/api/health", a.healthHandler)
	router.GET("/api/schema", a.schemaHandler)
	router.GET("/api/openapi.json", func(c *gin.Context) {
		c.JSON(http.StatusOK, openAPISpec())
	})
	router.GET("/json/stats.json", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, a.SnapshotStats())
	})

	api := router.Group("/api", a.authMiddleware())
	api.GET("/config", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "config": a.ConfigSnapshot()})
	})
	api.PUT("/config", func(c *gin.Context) {
		body, apiErr := decodeRequestObject(c)
		if apiErr != nil {
			a.writeAPIError(c, apiErr)
			return
		}
		doc, apiErr := a.ReplaceConfig(ConfigDocument(body))
		if apiErr != nil {
			a.writeAPIError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "reloaded": true, "pid": os.Getpid(), "generation": a.generation.Load(), "config": doc})
	})

	for _, collection := range []string{"servers", "monitors", "sslcerts", "watchdog"} {
		key := collection
		api.GET("/"+key, func(c *gin.Context) { a.getCollectionHandler(c, key) })
		api.POST("/"+key, func(c *gin.Context) { a.createCollectionHandler(c, key) })
		api.PUT("/"+key+"/:id", func(c *gin.Context) { a.updateCollectionHandler(c, key) })
		api.DELETE("/"+key+"/:id", func(c *gin.Context) { a.deleteCollectionHandler(c, key) })
	}
	api.POST("/servers/:id/reset-traffic", a.resetTrafficHandler)
	api.POST("/reload", func(c *gin.Context) {
		if apiErr := a.ReloadConfig(); apiErr != nil {
			a.writeAPIError(c, apiErr)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "operation": "reload", "pid": os.Getpid(), "generation": a.generation.Load()})
	})
	api.POST("/restart", func(c *gin.Context) {
		if apiErr := a.ReloadConfig(); apiErr != nil {
			a.writeAPIError(c, apiErr)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"ok": true, "operation": "restart", "mode": "in-process", "pid": os.Getpid(), "generation": a.generation.Load()})
	})

	router.NoRoute(a.staticHandler)
	return router
}

func (a *App) securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("Referrer-Policy", "same-origin")
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Header("Cache-Control", "no-store")
		}
		c.Next()
	}
}

func (a *App) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.opts.CORSOrigin != "" {
			c.Header("Access-Control-Allow-Origin", a.opts.CORSOrigin)
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Admin-Token")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions && strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (a *App) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.opts.AdminToken == "" {
			a.writeAPIError(c, &APIError{Status: 503, Message: "management API is disabled; set ADMIN_TOKEN to enable it"})
			c.Abort()
			return
		}
		token := ""
		authorization := c.GetHeader("Authorization")
		if len(authorization) >= 7 && strings.EqualFold(authorization[:7], "bearer ") {
			token = strings.TrimSpace(authorization[7:])
		}
		if token == "" {
			token = strings.TrimSpace(c.GetHeader("X-Admin-Token"))
		}
		if len(token) != len(a.opts.AdminToken) || subtle.ConstantTimeCompare([]byte(token), []byte(a.opts.AdminToken)) != 1 {
			a.writeAPIError(c, &APIError{Status: 401, Message: "invalid or missing admin token"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *App) healthHandler(c *gin.Context) {
	running := a.agentRunning.Load()
	c.JSON(http.StatusOK, gin.H{
		"ok": true, "enabled": a.opts.AdminToken != "",
		"service":    gin.H{"running": true, "pid": os.Getpid(), "version": version, "uptime": int64(time.Since(a.startedAt).Seconds()), "generation": a.generation.Load()},
		"agent":      gin.H{"running": running, "address": a.opts.AgentAddr},
		"configPath": a.opts.ConfigPath,
	})
}

func (a *App) schemaHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "schema": apiSchema(a.opts.AdminToken != "")})
}

func apiSchema(enabled bool) map[string]any {
	endpoints := []any{
		map[string]any{"method": "GET", "path": "/api/health", "auth": false},
		map[string]any{"method": "GET", "path": "/api/schema", "auth": false},
		map[string]any{"method": "GET", "path": "/api/openapi.json", "auth": false},
		map[string]any{"method": "GET", "path": "/api/config", "auth": true},
		map[string]any{"method": "PUT", "path": "/api/config", "auth": true, "body": "full config JSON"},
	}
	for _, key := range []string{"servers", "monitors", "sslcerts", "watchdog"} {
		spec := collectionSpecs[key]
		endpoints = append(endpoints,
			map[string]any{"method": "GET", "path": "/api/" + key, "auth": true},
			map[string]any{"method": "POST", "path": "/api/" + key, "auth": true, "body": spec.itemName + " JSON"},
			map[string]any{"method": "PUT", "path": "/api/" + key + "/{id}", "auth": true, "body": spec.itemName + " JSON"},
			map[string]any{"method": "DELETE", "path": "/api/" + key + "/{id}", "auth": true},
		)
	}
	endpoints = append(endpoints,
		map[string]any{"method": "POST", "path": "/api/servers/{username}/reset-traffic", "auth": true},
		map[string]any{"method": "POST", "path": "/api/reload", "auth": true},
		map[string]any{"method": "POST", "path": "/api/restart", "auth": true},
	)
	collections := make(map[string]any)
	for key, spec := range collectionSpecs {
		collections[key] = map[string]any{"item": spec.itemName, "idField": spec.idField, "required": spec.required, "optional": spec.optional}
	}
	return map[string]any{
		"version":   version,
		"auth":      map[string]any{"type": "bearer", "header": "Authorization: Bearer <ADMIN_TOKEN>", "enabled": enabled},
		"endpoints": endpoints, "collections": collections,
	}
}

func decodeRequestObject(c *gin.Context) (map[string]any, *APIError) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBody)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "request body too large") {
			status = http.StatusRequestEntityTooLarge
		}
		return nil, &APIError{Status: status, Message: "invalid JSON body", Details: map[string]any{"error": err.Error()}}
	}
	if object == nil {
		return nil, &APIError{Status: 400, Message: "request body must be an object"}
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, &APIError{Status: 400, Message: "request body must contain one JSON value"}
	}
	return object, nil
}

func (a *App) getCollectionHandler(c *gin.Context, key string) {
	doc := a.ConfigSnapshot()
	items, _ := doc[key].([]any)
	c.JSON(http.StatusOK, gin.H{"ok": true, key: items})
}

func (a *App) createCollectionHandler(c *gin.Context, key string) {
	item, apiErr := decodeRequestObject(c)
	if apiErr != nil {
		a.writeAPIError(c, apiErr)
		return
	}
	doc, apiErr := a.MutateConfig(func(doc ConfigDocument) *APIError {
		items, _ := doc[key].([]any)
		doc[key] = append(items, item)
		return nil
	})
	if apiErr != nil {
		a.writeAPIError(c, apiErr)
		return
	}
	items, _ := doc[key].([]any)
	created := items[len(items)-1]
	c.JSON(http.StatusCreated, gin.H{"ok": true, collectionSpecs[key].itemName: created, "reloaded": true, "pid": os.Getpid(), "config": doc})
}

func (a *App) updateCollectionHandler(c *gin.Context, key string) {
	item, apiErr := decodeRequestObject(c)
	if apiErr != nil {
		a.writeAPIError(c, apiErr)
		return
	}
	id := c.Param("id")
	if id == "" {
		a.writeAPIError(c, &APIError{Status: 400, Message: "item id is required"})
		return
	}
	var updated any
	doc, apiErr := a.MutateConfig(func(doc ConfigDocument) *APIError {
		items, _ := doc[key].([]any)
		index, _, findErr := findCollectionItem(items, collectionSpecs[key].idField, id, key != "servers")
		if findErr != nil {
			return findErr
		}
		if index < 0 {
			return &APIError{Status: 404, Message: collectionSpecs[key].itemName + " was not found", Details: map[string]any{"id": id}}
		}
		items[index] = item
		doc[key] = items
		updated = item
		return nil
	})
	if apiErr != nil {
		a.writeAPIError(c, apiErr)
		return
	}
	// Return the normalized item from the resulting document.
	items, _ := doc[key].([]any)
	if itemMap, ok := updated.(map[string]any); ok {
		idValue := fmt.Sprint(itemMap[collectionSpecs[key].idField])
		if index, normalized, _ := findCollectionItem(items, collectionSpecs[key].idField, idValue, key != "servers"); index >= 0 {
			updated = normalized
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, collectionSpecs[key].itemName: updated, "reloaded": true, "pid": os.Getpid(), "config": doc})
}

func (a *App) deleteCollectionHandler(c *gin.Context, key string) {
	id := c.Param("id")
	if id == "" {
		a.writeAPIError(c, &APIError{Status: 400, Message: "item id is required"})
		return
	}
	var removed any
	doc, apiErr := a.MutateConfig(func(doc ConfigDocument) *APIError {
		items, _ := doc[key].([]any)
		index, item, findErr := findCollectionItem(items, collectionSpecs[key].idField, id, key != "servers")
		if findErr != nil {
			return findErr
		}
		if index < 0 {
			return &APIError{Status: 404, Message: collectionSpecs[key].itemName + " was not found", Details: map[string]any{"id": id}}
		}
		removed = item
		doc[key] = append(items[:index], items[index+1:]...)
		return nil
	})
	if apiErr != nil {
		a.writeAPIError(c, apiErr)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "removed": removed, "reloaded": true, "pid": os.Getpid(), "config": doc})
}

func findCollectionItem(items []any, idField, id string, allowIndex bool) (int, any, *APIError) {
	if numeric, err := strconv.Atoi(id); allowIndex && err == nil {
		if numeric >= 0 && numeric < len(items) {
			return numeric, items[numeric], nil
		}
		return -1, nil, nil
	}
	matches := make([]int, 0, 1)
	for index, raw := range items {
		item, ok := raw.(map[string]any)
		if ok && fmt.Sprint(item[idField]) == id {
			matches = append(matches, index)
		}
	}
	if len(matches) > 1 {
		return -1, nil, &APIError{Status: 409, Message: "collection has duplicate " + idField + "; use numeric index instead", Details: map[string]any{"id": id}}
	}
	if len(matches) == 1 {
		return matches[0], items[matches[0]], nil
	}
	return -1, nil, nil
}

func (a *App) resetTrafficHandler(c *gin.Context) {
	username := c.Param("id")
	if username == "" {
		a.writeAPIError(c, &APIError{Status: 400, Message: "username is required"})
		return
	}
	result, apiErr := a.ResetTraffic(username)
	if apiErr != nil {
		a.writeAPIError(c, apiErr)
		return
	}
	result["ok"] = true
	result["operation"] = "reset-traffic"
	result["pid"] = os.Getpid()
	c.JSON(http.StatusOK, result)
}

func (a *App) writeAPIError(c *gin.Context, apiErr *APIError) {
	payload := gin.H{"ok": false, "error": apiErr.Message}
	if apiErr.Details != nil {
		payload["details"] = apiErr.Details
	}
	c.JSON(apiErr.Status, payload)
}

func (a *App) staticHandler(c *gin.Context) {
	if strings.HasPrefix(c.Request.URL.Path, "/api/") || strings.HasPrefix(c.Request.URL.Path, "/json/") {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "endpoint was not found"})
		return
	}
	requestPath := filepath.Clean("/" + c.Request.URL.Path)
	if requestPath == "/" {
		requestPath = "/index.html"
	}
	root, err := filepath.Abs(a.opts.WebDir)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	filePath := filepath.Join(root, strings.TrimPrefix(requestPath, "/"))
	if filePath != root && !strings.HasPrefix(filePath, root+string(os.PathSeparator)) {
		c.Status(http.StatusNotFound)
		return
	}
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}
	if contentType := mime.TypeByExtension(filepath.Ext(filePath)); contentType != "" {
		c.Header("Content-Type", contentType)
	}
	if filepath.Base(filePath) == "index.html" {
		c.Header("Cache-Control", "no-cache")
	}
	http.ServeFile(c.Writer, c.Request, filePath)
}
