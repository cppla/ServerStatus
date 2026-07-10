package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
)

func readConfig(path string) (ConfigDocument, RuntimeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, RuntimeConfig{}, err
	}
	doc, err := decodeDocument(data)
	if err != nil {
		return nil, RuntimeConfig{}, fmt.Errorf("parse %s: %w", path, err)
	}
	normalized, runtime, apiErr := normalizeConfig(doc)
	if apiErr != nil {
		return nil, RuntimeConfig{}, apiErr
	}
	return normalized, runtime, nil
}

func marshalIndented(value any) ([]byte, error) {
	data, err := jsonMarshalIndent(value)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// jsonMarshalIndent is a variable so file-writing failure paths can be tested.
var jsonMarshalIndent = func(value any) ([]byte, error) {
	return json.MarshalIndent(value, "", "\t")
}

func writeConfig(path string, doc ConfigDocument) error {
	data, err := marshalIndented(doc)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
		backup := fmt.Sprintf("%s.bak-%s", path, time.Now().Format("20060102-150405.000000000"))
		if err := copyFile(path, backup, mode); err != nil {
			return fmt.Errorf("backup config: %w", err)
		}
		pruneBackups(path, 10)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	return atomicWrite(path, data, mode, true)
}

func writeStatsFile(path string, value any) error {
	data, err := marshalIndented(value)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return atomicWrite(path, data, 0o644, false)
}

func atomicWrite(path string, data []byte, mode os.FileMode, allowBusyFallback bool) error {
	directory := filepath.Dir(path)
	tmp, err := os.CreateTemp(directory, ".serverstatus-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err == nil {
		return syncDirectory(directory)
	} else if !allowBusyFallback || !errors.Is(err, syscall.EBUSY) {
		return err
	}

	// Docker cannot rename over a single-file bind mount. The backup above is
	// already durable, so truncate and sync the mounted inode as a fallback.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func copyFile(source, destination string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err == nil {
		err = out.Sync()
	}
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func pruneBackups(configPath string, keep int) {
	matches, err := filepath.Glob(configPath + ".bak-*")
	if err != nil || len(matches) <= keep {
		return
	}
	sort.Strings(matches)
	for _, path := range matches[:len(matches)-keep] {
		_ = os.Remove(path)
	}
}

func syncDirectory(directory string) error {
	dir, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
