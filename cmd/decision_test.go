// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/testutil"
	"github.com/greycoderk/lore/internal/ui"
)

func TestRunCalibration_NotInitialized(t *testing.T) {
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	restore := ui.SaveAndDisableColor()
	defer restore()

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	err := runCalibration(streams, nil)
	if err == nil {
		t.Fatal("expected error for uninitialized repo")
	}
	if !strings.Contains(errBuf.String(), "Lore not initialized") {
		t.Errorf("expected 'Lore not initialized' in stderr, got: %q", errBuf.String())
	}
}

func TestRunCalibration_NilStore(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	restore := ui.SaveAndDisableColor()
	defer restore()

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	// nil storePtr
	err := runCalibration(streams, nil)
	if err == nil {
		t.Fatal("expected error for nil store")
	}
	if !strings.Contains(err.Error(), "unavail") && !strings.Contains(err.Error(), "store") {
		t.Errorf("error = %q, want store unavailable message", err)
	}
}

func TestRunCalibration_NilStoreValue(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	restore := ui.SaveAndDisableColor()
	defer restore()

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	// storePtr points to nil
	var s domain.LoreStore
	err := runCalibration(streams, &s)
	if err == nil {
		t.Fatal("expected error for nil store value")
	}
}

func TestDecisionCmd_CalibrationNotInitialized(t *testing.T) {
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cfg := &config.Config{}
	var s domain.LoreStore
	cmd := newDecisionCmd(cfg, streams, &s)
	cmd.SetArgs([]string{"--calibration"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for uninitialized repo")
	}
	if !strings.Contains(errBuf.String(), "Lore not initialized") {
		t.Errorf("expected 'Lore not initialized' in stderr, got: %q", errBuf.String())
	}
}

func TestDecisionCmd_CalibrationNilStore(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	cfg := &config.Config{}
	var s domain.LoreStore
	cmd := newDecisionCmd(cfg, streams, &s)
	cmd.SetArgs([]string{"--calibration"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}

func TestDecisionCmd_HasFlags(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := &config.Config{}
	var s domain.LoreStore
	cmd := newDecisionCmd(cfg, streams, &s)

	if cmd.Use != "decision" {
		t.Errorf("Use = %q, want %q", cmd.Use, "decision")
	}

	explainFlag := cmd.Flag("explain")
	if explainFlag == nil {
		t.Error("expected --explain flag")
	}

	calibrationFlag := cmd.Flag("calibration")
	if calibrationFlag == nil {
		t.Error("expected --calibration flag")
	}
}
