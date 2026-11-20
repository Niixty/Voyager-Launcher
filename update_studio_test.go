//go:build windows

package main

import (
    "io"
    "strings"
    "testing"
)

func TestPreviewDoesNotProbeNetwork(t *testing.T) {
    prior:=version
    version="0.9.0-preview"
    defer func(){version=prior}()
    result,err:=probeUpdateStatus()
    if err!=nil {t.Fatal(err)}
    if !result.Preview || result.Downloading {t.Fatalf("invalid preview result: %+v",result)}
}

func TestNormalCloseNeverInstallsWithoutConsent(t *testing.T) {
    updateState.Lock()
    oldPending,oldApproved:=updateState.pending,updateState.approved
    updateState.pending=&pendingUpdate{Version:"v9.9.9",File:"unavailable"}
    updateState.approved=false
    updateState.Unlock()
    defer func(){
        updateState.Lock()
        updateState.pending,updateState.approved=oldPending,oldApproved
        updateState.Unlock()
    }()
    if err:=launchPendingUpdate();err!=nil {t.Fatalf("unexpected update on normal close: %v",err)}
}

func TestDownloadProgressNeverExceeds94BeforeVerification(t *testing.T) {
    updateState.Lock()
    old:=updateState.progress
    updateState.progress=0
    updateState.Unlock()
    defer func(){updateState.Lock();updateState.progress=old;updateState.Unlock()}()
    reader:=&updateDownloadMeter{source:strings.NewReader("abcdefghij"),total:10}
    if _,err:=io.Copy(io.Discard,reader);err!=nil {t.Fatal(err)}
    updateState.Lock()
    progress:=updateState.progress
    updateState.Unlock()
    if progress!=94 {t.Fatalf("expected 94%% download progress, got %d%%",progress)}
}
