//go:build windows

package main

import (
    "strings"
    "testing"
)
func TestNewClassicRegistrationNeverReusesExistingID(t *testing.T) {
    used:=map[string]bool{"novojogo":true,"novojogo2":true,"novojogo3":true}
    id:=nextAvailableClassicGameID(func(id string)bool {return used[strings.ToLower(id)]})
    if id!="NovoJogo4" {t.Fatalf("o cadastro sobrescreveria jogo existente: %s",id)}
    id=nextAvailableClassicGameID(func(id string)bool{return false})
    if id!="NovoJogo" {t.Fatalf("ID inicial incorreto: %s",id)}
}
